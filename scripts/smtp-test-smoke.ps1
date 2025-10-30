$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$base = 'http://localhost:18081'

function Invoke-RestJson {
  param(
    [Parameter(Mandatory=$true)][string]$Method,
    [Parameter(Mandatory=$true)][string]$Uri,
    [hashtable]$Headers,
    $Body
  )
  try {
    $hdrs = @{ 'Accept-Encoding' = 'identity' }
    if ($Headers) { foreach ($k in $Headers.Keys) { $hdrs[$k] = $Headers[$k] } }
    if ($PSBoundParameters.ContainsKey('Body')) {
      return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $hdrs -ContentType 'application/json' -Body $Body
    } else {
      return Invoke-RestMethod -Method $Method -Uri $Uri -Headers $hdrs
    }
  } catch {
    if ($_.Exception.Response) {
      try { $sr = New-Object IO.StreamReader($_.Exception.Response.GetResponseStream()); $txt = $sr.ReadToEnd(); Write-Host "HTTP ERROR BODY:"; Write-Host $txt } catch {}
    }
    throw
  }
}
# Register/login
$rand = Get-Random -Maximum 1000000
$email = "smtp$rand@example.com"
$register = @{ full_name = 'SMTP User'; email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
Invoke-RestJson -Method Post -Uri "$base/auth/register" -Body $register | Out-Null
$loginBody = @{ email = $email; password = 'P@ssw0rd12345!' } | ConvertTo-Json
$loginRes = Invoke-RestJson -Method Post -Uri "$base/auth/login" -Body $loginBody
$token = $loginRes.token

# Call test-smtp
$body = @{ to = 'you@example.com' } | ConvertTo-Json
try {
  $res = Invoke-RestJson -Method Post -Uri "$base/admin/test-smtp" -Headers @{ Authorization = "Bearer $token" } -Body $body
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
