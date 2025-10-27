# Backend smoke test script
$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'

# Health checks
Write-Host "Healthz:"; (Invoke-WebRequest "$base/healthz").StatusCode
Write-Host "OpenAPI:"; (Invoke-WebRequest "$base/openapi.json").StatusCode

# Register
$rand = Get-Random -Maximum 1000000
$email = "dev$rand@example.com"
$register = @{ full_name = 'Dev User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$regRes = Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register
Write-Host "Registered:" ($regRes.email)

# Login
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token
Write-Host "Token acquired: " ($token.Substring(0,16) + '...')

# Get org
$orgs = Invoke-RestMethod -Method Get -Uri "$base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id
Write-Host "Org:" $orgId

# Create agent
$agentBody = @{ name = 'demo-agent'; description = 'demo agent' } | ConvertTo-Json
$agentRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $agentBody
$agentId = $agentRes.id
Write-Host "Agent:" $agentId

# Add allow rule (as nested JSON object, not a JSON string)
$ruleObj = @{ action = 'deploy'; effect = 'allow'; context = @{ env = 'prod' } }
$ruleBody = @{ rule = $ruleObj } | ConvertTo-Json -Depth 5 -Compress
$ruleRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/agents/$agentId/permissions" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $ruleBody
Write-Host "Rule:" $ruleRes.id

# Create API key
$keyBody = @{ name = 'dev key' } | ConvertTo-Json
$keyRes = Invoke-RestMethod -Method Post -Uri "$base/organizations/$orgId/apikeys" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $keyBody
$secret = $keyRes.secret_key
Write-Host "API key prefix:" $keyRes.key_prefix

# Verify with API key
$ctxObj = @{ action = 'deploy'; env = 'prod' }
$verifyObj = @{ agent_id = "$agentId"; request_context = $ctxObj }
$verifyJson = $verifyObj | ConvertTo-Json -Depth 5 -Compress
$verifyRes = Invoke-RestMethod -Method Post -Uri "$base/v1/verify" -Headers @{ 'X-API-Key' = $secret; 'AURA-Version' = '2025-10-01' } -ContentType 'application/json' -Body $verifyJson
Write-Host "Decision:" $verifyRes.decision "Reason:" $verifyRes.reason
