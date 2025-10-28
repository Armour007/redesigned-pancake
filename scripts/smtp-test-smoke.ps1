$ErrorActionPreference = 'Stop'
$base = 'http://localhost:8081'
# Register/login
$rand = Get-Random -Maximum 1000000
$email = "smtp$rand@example.com"
$register = @{ full_name = 'SMTP User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestMethod -Method Post -Uri "$base/auth/register" -ContentType 'application/json' -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestMethod -Method Post -Uri "$base/auth/login" -ContentType 'application/json' -Body $loginBody
$token = $loginRes.token

# Call test-smtp
$body = @{ to = 'you@example.com' } | ConvertTo-Json
try {
  $res = Invoke-RestMethod -Method Post -Uri "$base/admin/test-smtp" -Headers @{ Authorization = "Bearer $token" } -ContentType 'application/json' -Body $body -ErrorAction Stop
  Write-Host "SMTP ok:" ($res.ok)
} catch {
  if ($_.Exception.Response) {
    $sr = New-Object IO.StreamReader($_.Exception.Response.GetResponseStream())
    $txt = $sr.ReadToEnd()
    Write-Host $txt
  } else {
    Write-Host 'request failed'
  }
}
