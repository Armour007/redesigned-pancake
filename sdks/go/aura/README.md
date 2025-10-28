# AURA Go SDK

Minimal client, HTTP middleware, and webhook verifier for AURA in Go.

## 1‑minute plug‑in: HTTP middleware

Protect any handler with a single middleware. The adapter calls AURA and only forwards the request when allowed.

```go
package main

import (
	"net/http"
	"os"
	aura "github.com/Armour007/aura/sdks/go/aura"
)

func main() {
	client := aura.NewClient(
		os.Getenv("AURA_API_KEY"),
		getenv("AURA_API_BASE_URL", "http://localhost:8081"),
		os.Getenv("AURA_VERSION"),
	)
	agentID := os.Getenv("AURA_AGENT_ID")

	mux := http.NewServeMux()
	mux.Handle("/danger", aura.ProtectHTTP(agentID, client, func(r *http.Request) any {
		return map[string]any{"path": r.URL.Path, "method": r.Method}
	}, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})))

	http.ListenAndServe(":3000", mux)
}
```

## Client usage

```go
c := aura.NewClient(os.Getenv("AURA_API_KEY"), os.Getenv("AURA_API_BASE_URL"), os.Getenv("AURA_VERSION"))
resp, err := c.Verify(os.Getenv("AURA_AGENT_ID"), map[string]any{"action": "deploy:prod"})
```

### Webhooks

```go
ok, err := aura.VerifySignature(os.Getenv("AURA_WEBHOOK_SECRET"), r.Header.Get("AURA-Signature"), body, 0)
if !ok { w.WriteHeader(401) }
```

## Module path
This SDK lives inside the monorepo under `sdks/go/aura`. Consumers can import it using the full path:

```
import "github.com/Armour007/aura/sdks/go/aura"
```

Optionally, use go.work or replace directives if developing locally.
