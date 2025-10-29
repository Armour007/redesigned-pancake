package mesh

import "fmt"

// NewNatsBus default stub for builds without 'nats' tag
func NewNatsBus(url string) (Bus, error) {
	return nil, fmt.Errorf("nats backend not available: build without -tags nats")
}
