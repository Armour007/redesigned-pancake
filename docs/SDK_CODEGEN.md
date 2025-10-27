# SDK Code Generation (Multi-language)

We ship hand‑curated SDKs for Node, Python, and Go, and generate additional SDKs for Java, C#, Ruby, PHP, Rust, Swift, Kotlin, Dart, and C++ using OpenAPI Generator.

## Prerequisites
- Docker Desktop installed and running
- Backend available to fetch openapi.json (or generate once and commit)

## Generate SDKs

Windows PowerShell:

```
cd sdks/codegen
# Optional: point to a different backend URL
# .\generate.ps1 -OpenApiUrl "http://localhost:8081/openapi.json"
.\u005cgenerate.ps1
```

Outputs:
- sdks/java, sdks/csharp, sdks/ruby, sdks/php, sdks/rust, sdks/swift, sdks/kotlin, sdks/dart, sdks/cpp

## Notes
- Publish to registries when ready: Maven Central (Java), NuGet (C#), crates.io (Rust), etc.
- Keep the REST contract stable; SDKs are thin clients over /v1/verify and auth.
- Provide a small HMAC signature helper for webhook verification in each language.
