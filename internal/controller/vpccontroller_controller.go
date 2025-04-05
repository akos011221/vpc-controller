/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	vpcv1alpha1 "akosrbn.io/vpccontroller/api/v1alpha1"
)

// Definitions to manage status conditions
const (
	typeReadyVPCController = "Ready"
)

// VPCControllerReconciler reconciles a VPCController object
type VPCControllerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vpc.akosrbn.io,resources=vpccontrollers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vpc.akosrbn.io,resources=vpccontrollers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vpc.akosrbn.io,resources=vpccontrollers/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VPCController object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.4/pkg/reconcile
func (r *VPCControllerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the VPCController instance, this checks if the Custom Resource (CR) for VPCController
	// applied on the cluster. If not, we can stop the reconciliation.
	vpcController := &vpcv1alpha1.VPCController{}
	err := r.Get(ctx, req.NamespacedName, vpcController)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If not found, it is either deleted or never existed
			// In this case, we can stop the reconciliation
			log.Info("VPCController resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get VPCController")
		return ctrl.Result{}, err
	}

	// Set the status as Unknown when no status is available
	if len(vpcController.Status.Conditions) == 0 {
		meta.SetStatusCondition(&vpcController.Status.Conditions, metav1.Condition{
			Type:    typeReadyVPCController,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err = r.Status().Update(ctx, vpcController); err != nil {
			log.Error(err, "Failed to update VPCController status")
			return ctrl.Result{}, err
		}

		// Re-fetch the VPCController Custom Resource after updating the
		// status so that we have the latest state of the resource on the
		// cluster and we will avoid raising the error "the object has been
		// modified, please apply your changes to the latest version and try
		// again" which would re-trigger the reconciliation if we try to update
		// it again the following operations
		if err = r.Get(ctx, req.NamespacedName, vpcController); err != nil {
			log.Error(err, "Failed to re-fetch VPCController after status update")
			return ctrl.Result{}, err
		}
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      vpcController.Name,
		Namespace: vpcController.Namespace,
	}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForVPCController(vpcController)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for VPCController")

			// Update the status
			meta.SetStatusCondition(&vpcController.Status.Conditions, metav1.Condition{
				Type:    typeReadyVPCController,
				Status:  metav1.ConditionFalse,
				Reason:  "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", vpcController.Name, err),
			})

			if err := r.Status().Update(ctx, vpcController); err != nil {
				log.Error(err, "Failed to update VPCController status")
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace,
			"Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace,
				"Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment creates successfully
		// Requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Return an error so the reconciliation will
		// be re-triggered again
		return ctrl.Result{}, err
	}

	// The CRD API defines that the VPCController type have a VPCControllerSpec.Size
	// field to set the quantity of Deployment instances to the desired state on the
	// cluster. Therefore, the following code will ensure that the Deployment size
	// is the same as defined via the Size spec of the Custom Resource which we are
	// reconciling.
	size := vpcController.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace,
				"Deployment.Name", found.Name)

			// Re-fetch the VPCController Custom Resource before
			// updating the status so that we have the latest state
			// of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please
			// apply your changes to the latest version and try again"
			// which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, vpcController); err != nil {
				log.Error(err, "Failed to re-fetch VPCController")
				return ctrl.Result{}, err
			}

			// Update the status
			meta.SetStatusCondition(&vpcController.Status.Conditions, metav1.Condition{
				Type:    typeReadyVPCController,
				Status:  metav1.ConditionFalse,
				Reason:  "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", vpcController.Name, err),
			})
			if err := r.Status().Update(ctx, vpcController); err != nil {
				log.Error(err, "Failed to update VPCController status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource
		// before update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following will update the status
	meta.SetStatusCondition(&vpcController.Status.Conditions, metav1.Condition{
		Type:    typeReadyVPCController,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", vpcController.Name, size),
	})

	if err := r.Status().Update(ctx, vpcController); err != nil {
		log.Error(err, "Failed to update VPCController status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VPCControllerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vpcv1alpha1.VPCController{}).
		Named("vpccontroller").
		Complete(r)
}

// deploymentForVPCController returns a VPCController Deployment object
func (r *VPCControllerReconciler) deploymentForVPCController(
	vpcController *vpcv1alpha1.VPCController) (*appsv1.Deployment, error) {
	replicas := vpcController.Spec.Size
	image := "vpccontroller:v0.1.0"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpcController.Name,
			Namespace: vpcController.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": "vpccontroller"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "vpccontroller"},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "vpccontroller",
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             ptr.To(true),
							RunAsUser:                ptr.To(int64(3333)),
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 12000,
							Name:          "vpccontroller",
						}},
						Command: []string{"vpccontroller", "run"},
					}},
				},
			},
		},
	}

	// Set the owner reference for the deployment
	if err := ctrl.SetControllerReference(vpcController, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}
