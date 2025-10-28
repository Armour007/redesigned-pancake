$ErrorActionPreference = 'Stop'
Set-Location "$PSScriptRoot/../backend"
$env:PORT = '8081'
$env:AURA_FRONTEND_BASE_URL = 'http://localhost:5173'
$env:AURA_API_BASE_URL = 'http://localhost:8081'
$env:AURA_DOWNLOAD_SIGNING_KEY = 'dev_signing_key_123'
go run ./cmd/server
