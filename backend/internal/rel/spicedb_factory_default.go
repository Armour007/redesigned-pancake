package rel

import "fmt"

// NewSpiceDBFromEnv is a default (no-tag) stub returning an error to indicate SpiceDB is not enabled in this build.
// The spicedb-tagged variant provides the real implementation.
func NewSpiceDBFromEnv(endpoint, token string) (GraphClient, error) {
	return nil, fmt.Errorf("spicedb backend not available: build without -tags spicedb")
}
