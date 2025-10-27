param(
  [string]$OpenApiUrl = "http://localhost:8081/openapi.json"
)
$ErrorActionPreference = 'Stop'

# Paths
$repoRoot = Resolve-Path (Join-Path $PSScriptRoot '..\..')
$specPath = Join-Path $repoRoot 'backend\static\openapi.json'

Write-Host "Fetching OpenAPI spec from $OpenApiUrl ..."
Invoke-WebRequest -Uri $OpenApiUrl -OutFile $specPath -UseBasicParsing

# Ensure Docker is available
$docker = (Get-Command docker -ErrorAction SilentlyContinue)
if(-not $docker){
  throw "Docker is required to run openapi-generator in a portable way. Please install Docker Desktop."
}

function Invoke-Codegen {
  param(
    [string]$Generator,
    [string]$OutputDir
  )
  Write-Host "Generating SDK: $Generator -> $OutputDir"
  docker run --rm -v "$repoRoot:/work" -w /work/sdks/codegen openapitools/openapi-generator-cli:latest generate `
    -c openapi-generator.yaml `
    -g $Generator `
    -o $OutputDir
}

# Explicit generation per language to keep PS 5.1 compatible
Invoke-Codegen -Generator 'java' -OutputDir '../java'
Invoke-Codegen -Generator 'csharp' -OutputDir '../csharp'
Invoke-Codegen -Generator 'ruby' -OutputDir '../ruby'
Invoke-Codegen -Generator 'php' -OutputDir '../php'
Invoke-Codegen -Generator 'rust' -OutputDir '../rust'
Invoke-Codegen -Generator 'swift5' -OutputDir '../swift'
Invoke-Codegen -Generator 'kotlin' -OutputDir '../kotlin'
Invoke-Codegen -Generator 'dart-dio' -OutputDir '../dart'
Invoke-Codegen -Generator 'cpp-httplib' -OutputDir '../cpp'

Write-Host "Done. SDKs generated under sdks/*"
