# SpiceDB quickstart for AURA

This guide shows how to run SpiceDB locally with Docker and apply a schema suitable for AURA's agents, users, teams, orgs, and resources.

## Run SpiceDB locally (Docker)

1. Start SpiceDB (no TLS, dev-only):

```powershell
# Windows PowerShell
$env= @{}
$CONTAINER_NAME = "spicedb"
$PORT = 50051

docker run --name $CONTAINER_NAME -p $PORT:50051 -e "SPICEDB_GRPC_PRESHARED_KEY=dev-secret" authzed/spicedb serve --grpc-preshared-key dev-secret --grpc-no-tls
```

2. Install `zhed` or use `spicedb`/`authzed` CLI to apply schema (pick one tool you prefer). For quick testing, you can use the `authzed` container:

```powershell
# Apply schema using authzed/zed in a container
$SCHEMA = (Get-Content -Raw "./docs/spicedb/schema.zed")

docker run --rm -i --network host authzed/zed:latest zed schema write --endpoint localhost:50051 --insecure --token dev-secret --schema - << EOF
$SCHEMA
EOF
```

3. Point AURA backend to SpiceDB:

- Build backend with SpiceDB support:

```powershell
# In backend directory
$env:GOFLAGS = "-tags=spicedb"
go build ./...
```

- Set env vars and run backend:

```powershell
$env:AURA_REL_BACKEND = "spicedb"
$env:AURA_SPICEDB_ENDPOINT = "localhost:50051"
$env:AURA_SPICEDB_TOKEN = "dev-secret"
# optional caching
$env:AURA_REL_CACHE_TTL_MS = "2000"
$env:AURA_REL_NEG_CACHE_TTL_MS = "500"

# run your usual backend task, e.g. VS Code task or:
# go run -tags=spicedb ./cmd/server
```

## Schema

The schema is in `docs/spicedb/schema.zed`. It defines basic namespaces and relations:

- `org`, `team`, `user`, `agent`, `resource`
- `member`, `can_act_for`, and resource permissions: `viewer`, `editor`, `owner`
- Relation implications: `owner` implies `editor`, and `editor` implies `viewer`.

You can evolve this schema as your needs grow.
