package sdk

// Registry provides service registration utilities
// This file is mainly for documentation purposes as registration
// is handled automatically by the Client

type Registry struct {
	client *Client
}

// Register is automatically called when creating a new Client
// No manual registration is needed
func (r *Registry) Register() error {
	// Registration is handled in client.connect()
	return nil
}

// Deregister is automatically called when closing the Client
// No manual deregistration is needed
func (r *Registry) Deregister() error {
	// Deregistration is handled in client.Close()
	return nil
}

// GetInstanceID returns the instance ID assigned by Anox
func (r *Registry) GetInstanceID() string {
	if r.client == nil {
		return ""
	}
	return r.client.GetInstanceID()
}
