param(
    [Parameter(Mandatory=$true)][string]$BaseUrl = "http://localhost:8081",
    [Parameter(Mandatory=$true)][string]$OrgId,
    [Parameter(Mandatory=$true)][string]$AgentId,
    [Parameter(Mandatory=$false)][string]$ApiKey,
    [Parameter(Mandatory=$false)][string]$AttestToken,
    [int]$Count = 30,
    [int]$IntervalMs = 500
)

# Build headers: prefer attestation Bearer, else API key
$headers = @{"Content-Type"="application/json"}
if ($AttestToken) {
  $headers["Authorization"] = "Bearer $AttestToken"
} elseif ($ApiKey) {
  $headers["X-API-Key"] = $ApiKey
} else {
  Write-Error "Provide -AttestToken or -ApiKey"
  exit 1
}

$body = @{ agent_id = $AgentId; request_context = @{ action = "read"; resource = "demo" } } | ConvertTo-Json -Depth 5

$verifyUrl = "$BaseUrl/v2/verify"
$signalsUrl = "$BaseUrl/v2/signals/risk?org_id=$OrgId&agent_id=$AgentId"

Write-Host "Hitting $verifyUrl $Count times every $IntervalMs ms..."
for ($i=1; $i -le $Count; $i++) {
  try {
    $resp = Invoke-RestMethod -Method POST -Uri $verifyUrl -Headers $headers -Body $body
    Write-Host ("{0:00}: allow={1} reason='{2}'" -f $i, $resp.allow, $resp.reason)
  } catch {
    Write-Warning $_
  }
  Start-Sleep -Milliseconds $IntervalMs
}

Write-Host "Fetching risk signals..."
try {
  $sig = Invoke-RestMethod -Method GET -Uri $signalsUrl -Headers $headers
  Write-Host ("risk.score={0} flags=[{1}]" -f $sig.risk.score, ($sig.risk.flags -join ","))
} catch { Write-Warning $_ }

Write-Host "Sending one more verify to observe deny if spike occurred..."
try {
  $resp = Invoke-RestMethod -Method POST -Uri $verifyUrl -Headers $headers -Body $body
  Write-Host ("final: allow={0} reason='{1}'" -f $resp.allow, $resp.reason)
} catch { Write-Warning $_ }
