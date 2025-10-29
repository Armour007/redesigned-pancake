param(
  [Parameter(Mandatory=$true)][string]$BaseUrl,
  [Parameter(Mandatory=$true)][string]$ApiKey,
  [Parameter(Mandatory=$true)][string]$OrgId,
  [Parameter(Mandatory=$true)][string]$PolicyId,
  [Parameter(Mandatory=$true)][string]$PolicyFile,
  [string]$ChangeTicket = "CI-${env:GITHUB_RUN_ID}-${env:GITHUB_RUN_ATTEMPT}"
)

$ErrorActionPreference = 'Stop'
$headers = @{ 'X-API-Key' = $ApiKey; 'Content-Type' = 'application/json' }

Write-Host "Adding new policy version..."
$body = @{ body = (Get-Content -Raw -Path $PolicyFile | ConvertFrom-Json) ; change_ticket = $ChangeTicket } | ConvertTo-Json -Depth 100
Invoke-RestMethod -Method Post -Uri "$BaseUrl/organizations/$OrgId/policies/$PolicyId/versions" -Headers $headers -Body $body | Out-Null

Write-Host "Approving policy version (threshold enforced by backend)..."
# Fetch latest version number
$vers = Invoke-RestMethod -Method Get -Uri "$BaseUrl/organizations/$OrgId/policies/$PolicyId/versions" -Headers $headers
$version = ($vers | Sort-Object -Property version -Descending | Select-Object -First 1).version
Invoke-RestMethod -Method Post -Uri "$BaseUrl/organizations/$OrgId/policies/$PolicyId/versions/$version/approve" -Headers $headers | Out-Null
Write-Host "Approval recorded for version $version"

Write-Host "Done"param(
  [Parameter(Mandatory=$true)][string]$BaseUrl,
  [Parameter(Mandatory=$true)][string]$ApiKey,
  [Parameter(Mandatory=$true)][string]$OrgId,
  [Parameter(Mandatory=$true)][string]$PolicyId,
  [Parameter(Mandatory=$true)][string]$PolicyFile
)

$headers = @{ 'X-API-Key' = $ApiKey; 'Content-Type' = 'application/json' }

if (-not (Test-Path $PolicyFile)) { Write-Error "Policy file not found: $PolicyFile"; exit 1 }
$body = Get-Content -Raw -Path $PolicyFile | ConvertFrom-Json | ConvertTo-Json -Compress

# Create a new version
$payload = @{ body = ($body | ConvertFrom-Json); change_ticket = "CI-${env:GITHUB_RUN_ID}-${env:GITHUB_SHA}" } | ConvertTo-Json -Compress
$version = Invoke-RestMethod -Method POST -Uri "$BaseUrl/organizations/$OrgId/policies/$PolicyId/versions" -Headers $headers -Body $payload

# Approve the version (approval threshold in backend will mark approved once reached)
Invoke-RestMethod -Method POST -Uri "$BaseUrl/organizations/$OrgId/policies/$PolicyId/versions/$($version.version)/approve" -Headers $headers

# Optionally activate after a dry-run simulation (skipped here)
Write-Host "Created and approved policy version $($version.version)"
