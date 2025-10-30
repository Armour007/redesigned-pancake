param(
  [string]$Base = "http://localhost:8081",
  [string]$ApiKey = "aura_sk_test",
  [string]$AgentId = "00000000-0000-0000-0000-000000000000",
  [int]$Rate = 50,
  [int]$DurationSeconds = 60
)

$body = '{"agent_id":"'+$AgentId+'","request_context":{"action":"read","resource":"doc:1"}}'
$target = "$Base/v1/verify"

$attack = "POST $target\nContent-Type: application/json\nX-API-Key: $ApiKey\nAURA-Version: 2025-10-01\n@ $body"

$attack | vegeta attack -rate "$Rate/s" -duration ${DurationSeconds}s | vegeta report
