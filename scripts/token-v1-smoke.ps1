param(
    [string]$ApiBase = $env:AURA_API_BASE_URL
)

if (-not $ApiBase) { $ApiBase = "http://localhost:8081" }
Write-Host "Using API base: $ApiBase"

# Optional: attempt to insert a local Ed25519 org trust key via psql if available
$OrgId = "org_smoke"
$psql = Get-Command psql -ErrorAction SilentlyContinue
if ($psql) {
    try {
        # Generate a random 32-byte seed as base64url in PowerShell
        $seed = New-Object byte[] 32
        [System.Security.Cryptography.RandomNumberGenerator]::Fill($seed)
        $seedB64Url = [Convert]::ToBase64String($seed).TrimEnd('=') -replace '\+','-' -replace '/','_'
        $kid = "smk_" + ([System.BitConverter]::ToString($seed)[0..3] -join '').ToLower()
        $sql = @"
INSERT INTO trust_keys(org_id, alg, ed25519_private_key_base64, kid, active, provider, key_ref, key_version, provider_config, jwk_pub, created_at)
VALUES ('$OrgId','EdDSA','$seedB64Url','$kid',true,'local',NULL,NULL,'{}'::jsonb,'{}'::jsonb,NOW())
ON CONFLICT DO NOTHING;
"@
        Write-Host "Attempting to insert local trust key via psql for $OrgId ..."
        $env:PGPASSWORD = $env:DB_PASSWORD
        $dbHost = $env:DB_HOST; if (-not $dbHost) { $dbHost = 'localhost' }
        $dbPort = $env:DB_PORT; if (-not $dbPort) { $dbPort = '5432' }
        $dbUser = $env:DB_USER; if (-not $dbUser) { $dbUser = 'aura_user' }
        $dbName = $env:DB_NAME; if (-not $dbName) { $dbName = 'aura_db' }
        $args = @('-h', $dbHost, '-p', $dbPort, '-U', $dbUser, '-d', $dbName, '-c', $sql)
        & $psql @args | Out-Null
    } catch { Write-Warning "psql insert failed: $_" }
} else {
    Write-Host "psql not found; skipping org trust key insert. Using env-based or HS256 fallback."
}

# Issue v1 token
$issueBody = @{ org_id = $OrgId; sub = "user_smoke"; aud = "svc"; action = "read"; resource = "doc:42"; ttl_sec = 300 } | ConvertTo-Json -Compress
$resp = Invoke-RestMethod -Method POST -Uri "$ApiBase/v1/token/issue" -Body $issueBody -ContentType 'application/json' -ErrorAction SilentlyContinue
if (-not $resp) { throw "Issue failed. Ensure backend is running at $ApiBase and env is configured." }
Write-Host "Issued token alg=$($resp.alg) kid=$($resp.kid) jti=$($resp.jti)"

# Verify v1 token
$verifyBody = @{ token = $resp.token } | ConvertTo-Json -Compress
$verify = Invoke-RestMethod -Method POST -Uri "$ApiBase/v1/token/verify" -Body $verifyBody -ContentType 'application/json'
Write-Host "Verify: valid=$($verify.valid) reason=$($verify.reason)"
if (-not $verify.valid) { throw "Verification failed: $($verify | ConvertTo-Json -Depth 6)" }

# Revoke token by JTI
$exp = [int64]([DateTimeOffset]::UtcNow.ToUnixTimeSeconds() + 300)
$revokeBody = @{ org_id = $OrgId; jti = $resp.jti; exp = $exp } | ConvertTo-Json -Compress
$rev = Invoke-RestMethod -Method POST -Uri "$ApiBase/v1/token/revoke" -Body $revokeBody -ContentType 'application/json'
Write-Host "Revoke: $($rev | ConvertTo-Json -Compress)"

# Introspect mark_used twice to assert replay detection
$intBody = @{ token = $resp.token; mark_used = $true } | ConvertTo-Json -Compress
$int1 = Invoke-RestMethod -Method POST -Uri "$ApiBase/v2/tokens/introspect" -Body $intBody -ContentType 'application/json'
$int2 = Invoke-RestMethod -Method POST -Uri "$ApiBase/v2/tokens/introspect" -Body $intBody -ContentType 'application/json'
Write-Host "Introspect1: valid=$($int1.valid) reason=$($int1.reason)"
Write-Host "Introspect2: valid=$($int2.valid) reason=$($int2.reason)"
if (-not $int1.valid -or $int2.valid) { throw "Replay detection failed." }

Write-Host "Token v1 smoke: OK"
