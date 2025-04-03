package models

// PolicyBasedRoute represents a GCP policy-based route entry.
type PolicyBasedRoute struct {
	Name        string `json:"name"`
	SourceAddr  string `json:"source_addr"`
	DestAddr    string `json:"dest_addr"`
	NextHopType string `json:"next_hop_type"`
	NextHop     string `json:"next_hop"`
}
