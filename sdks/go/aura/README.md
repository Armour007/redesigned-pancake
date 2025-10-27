# AURA Go SDK

Minimal client and webhook verifier for AURA in Go.

## Usage

```go
c := aura.NewClient(os.Getenv("AURA_API_KEY"), os.Getenv("AURA_API_BASE"), os.Getenv("AURA_VERSION"))
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
