# Creates an agent, allow rule, and API key, then prints the Quick Start URL.
param(
  [string]$Base = 'http://localhost:8081'
)
$ErrorActionPreference = 'Stop'

# Register + Login
$rand = Get-Random -Maximum 1000000
$email = "cli$rand@example.com"
$reg = @{ full_name = 'CLI User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$Base/auth/register" -ContentType 'application/json' -Body $reg | Out-Null
$login = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$Base/auth/login" -ContentType 'application/json' -Body $login
$token = $loginRes.token

# Org
$orgs = Invoke-RestMethod -Method Get -Uri "$Base/organizations/mine" -Headers @{ Authorization = "Bearer $token" }
$orgId = $orgs[0].id

# Agent
$agentBody = @{ name = 'cli-agent'; description = 'created by aura-quickstart' } | ConvertTo-Json
$agent = Invoke-RestMethod -Method Post -Uri "$Base/organizations/$orgId/agents" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $agentBody
$agentId = $agent.id

# Rule
$ruleObj = @{ action = 'deploy'; effect = 'allow'; context = @{ env = 'prod' } }
$ruleBody = @{ rule = $ruleObj } | ConvertTo-Json -Depth 5
Invoke-RestMethod -Method Post -Uri "$Base/organizations/$orgId/agents/$agentId/permissions" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $ruleBody | Out-Null

# API key
$keyBody = @{ name = 'cli key' } | ConvertTo-Json
$keyRes = Invoke-RestMethod -Method Post -Uri "$Base/organizations/$orgId/apikeys" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $keyBody
$keyPrefix = $keyRes.key_prefix
$quick = if($keyRes.quickstart_url){ $keyRes.quickstart_url + "&agent_id=$agentId" } else { "http://localhost:5173/quickstart?key_prefix=$keyPrefix&agent_id=$agentId" }

Write-Host "Agent:" $agentId
Write-Host "Key Prefix:" $keyPrefix
Write-Host "Open Quick Start:" $quick
