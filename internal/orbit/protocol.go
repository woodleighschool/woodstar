package orbit

// EnrollRequest is the JSON body Orbit POSTs to /api/fleet/orbit/enroll.
type EnrollRequest struct {
	EnrollSecret   string `json:"enroll_secret"`
	HardwareUUID   string `json:"hardware_uuid"`
	HardwareSerial string `json:"hardware_serial,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	ComputerName   string `json:"computer_name,omitempty"`
	HardwareModel  string `json:"hardware_model,omitempty"`
}

// EnrollResponse is the JSON body returned to a successful enrollment.
// orbit_node_key is the credential Orbit uses on subsequent calls.
type EnrollResponse struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

// ConfigRequest carries Orbit's node key.
type ConfigRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
}

// ConfigResponse is the Orbit config response.
type ConfigResponse struct{}

// DeviceMappingRequest carries a profile-provided email.
type DeviceMappingRequest struct {
	OrbitNodeKey string `json:"orbit_node_key"`
	Email        string `json:"email"`
}
