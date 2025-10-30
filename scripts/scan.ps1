param(
  [string]$RepoRoot = '.'
)
$ErrorActionPreference = 'Stop'
$root = Resolve-Path $RepoRoot
$files = Get-ChildItem -Recurse -File -Path $root | ForEach-Object {
  [PSCustomObject]@{
    path = $_.FullName.Substring($root.Path.Length + 1)
    size_bytes = [int64]$_.Length
  }
}
$large = $files | Where-Object { $_.size_bytes -gt 1MB }
$result = [PSCustomObject]@{
  generated_at = (Get-Date).ToString('o')
  repo_root = $root.Path
  total_files = $files.Count
  large_files = $large
}
$result | ConvertTo-Json -Depth 6
