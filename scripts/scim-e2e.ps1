# SCIM end-to-end smoke
$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'
$tok = $env:AURA_SCIM_TOKEN
if (-not $tok) { Write-Host 'Set AURA_SCIM_TOKEN'; exit 1 }

# Use existing user token and org from localStorage-equivalents? For script, register a temp user
$rand = Get-Random -Maximum 1000000
$email = "scim$rand@example.com"
$reg = @{ full_name = 'SCIM Admin'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$regRes = Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $reg
$login = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $login
$token = $loginRes.token
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id

# Create SCIM user
$uEmail = "user$rand@example.com"
$body = @{ userName = $uEmail; name = @{ givenName = 'User'; familyName = "$rand" }; groups = @(@{ display = 'member' }) } | ConvertTo-Json -Compress
$u = Invoke-RestMethod -Method Post -Uri "$base/scim/v2/Users?orgId=$orgId" -Headers @{ Authorization = "Bearer $tok" } -ContentType 'application/json' -Body $body
Write-Host "Created user" $u.id

# Deprovision via PATCH
$patch = @{ active = $false } | ConvertTo-Json
Invoke-RestMethod -Method Patch -Uri "$base/scim/v2/Users/$($u.id)?orgId=$orgId" -Headers @{ Authorization = "Bearer $tok" } -ContentType 'application/json' -Body $patch | Out-Null
Write-Host "Deprovisioned user"

# Reprovision via PATCH
$patch2 = @{ active = $true; role = 'read-only' } | ConvertTo-Json
Invoke-RestMethod -Method Patch -Uri "$base/scim/v2/Users/$($u.id)?orgId=$orgId" -Headers @{ Authorization = "Bearer $tok" } -ContentType 'application/json' -Body $patch2 | Out-Null
Write-Host "Reprovisioned user"

# List groups
$groups = Invoke-RestMethod -Method Get -Uri "$base/scim/v2/Groups?orgId=$orgId" -Headers @{ Authorization = "Bearer $tok" }
Write-Host "Groups:" ($groups.totalResults)
