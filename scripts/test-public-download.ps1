$ErrorActionPreference = 'SilentlyContinue'
try {
  $res = Invoke-WebRequest -UseBasicParsing -Method Get 'http://localhost:8081/sdk/public/download-generated/test'
  if ($res) { $res.StatusCode } else { 0 }
} catch {
  if ($_.Exception.Response) {
    [int]$_.Exception.Response.StatusCode.value__
  } else {
    -1
  }
}
