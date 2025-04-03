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

package v1

import (
	"github.com/akos011221/vpc-controller/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VPCControllerSpec defines the desired state of VPCController.
type VPCControllerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ControlledProjects []string                  `json:"controlled_projects"`
	PolicyBasedRoutes  []models.PolicyBasedRoute `json:"policy_based_routes"`
	SubnetMaxSize      int32                     `json:"subnet_max_size"`
}

// VPCControllerStatus defines the observed state of VPCController.
type VPCControllerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// VPCController is the Schema for the vpccontrollers API.
type VPCController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VPCControllerSpec   `json:"spec,omitempty"`
	Status VPCControllerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VPCControllerList contains a list of VPCController.
type VPCControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VPCController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VPCController{}, &VPCControllerList{})
}
