//go:build spicedb

package rel

// NewSpiceDBFromEnv constructs a SpiceDB GraphClient when spicedb build tag is enabled.
func NewSpiceDBFromEnv(endpoint, token string) (GraphClient, error) {
	return NewSpiceDBGraph(endpoint, token)
}
