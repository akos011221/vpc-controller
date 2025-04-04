package models

// PolicyBasedRoute represents a GCP policy-based route entry.
type PolicyBasedRoute struct {
	Name        string `json:"name"`
	SourceAddr  string `json:"source_addr"`
	DestAddr    string `json:"dest_addr"`
	NextHopType string `json:"next_hop_type"`
	NextHop     string `json:"next_hop"`
}

// GetPolicyBasedRoutes takes a project ID and a VPC id and returns a slice of PolicyBasedRoute
// objects from GCP data.
func GetPolicyBasedRoutes(projectID string, vpc string) ([]PolicyBasedRoute, error) {
	// Later, I'll add the logic here to get the PBRs from the project
	// using the GCP Go client library.
	return nil, nil
}

// CreatePolicyBasedRoute takes a project ID, a VPC id and a PolicyBasedRoute to create a new
// policy-based route in GCP.
func CreatePolicyBasedRoute(projectID string, vpc string, pbr PolicyBasedRoute) error {
	// Later, I'll add the logic here to create the PBR in GCP
	// using the GCP Go client library.
	return nil
}
