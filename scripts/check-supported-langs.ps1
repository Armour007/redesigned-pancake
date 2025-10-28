$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'
# Register and login
$rand = Get-Random -Maximum 1000000
$email = "langs$rand@example.com"
$register = @{ full_name = 'Langs User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token
# Call supported-langs
$res = Invoke-RestMethod -Method Get -Uri "$base/sdk/supported-langs" -Headers @{ Authorization = "Bearer $token" }
Write-Host "curated:" ($res.curated -join ',')
Write-Host "codegen:" ($res.codegen -join ',')
