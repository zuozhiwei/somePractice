$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
  Push-Location web
  try { npm ci --no-audit --no-fund; if ($LASTEXITCODE) { throw 'npm ci failed' }; npm run build; if ($LASTEXITCODE) { throw 'frontend build failed' } } finally { Pop-Location }
  $target = Join-Path $PSScriptRoot 'internal/frontend/dist'
  if (Test-Path -LiteralPath $target) {
    $resolved = (Resolve-Path -LiteralPath $target).Path
    $expected = [IO.Path]::GetFullPath($target)
    if ((Get-Item -LiteralPath $target -Force).Attributes -band [IO.FileAttributes]::ReparsePoint) { throw 'Refusing to remove a linked frontend directory' }
    if ($resolved -ne $expected -or -not $resolved.StartsWith([IO.Path]::GetFullPath($PSScriptRoot) + [IO.Path]::DirectorySeparatorChar, [StringComparison]::OrdinalIgnoreCase)) { throw 'Refusing to remove a frontend directory outside the project' }
    Remove-Item -LiteralPath $resolved -Recurse -Force
  }
  New-Item -ItemType Directory -Force -Path $target, bin | Out-Null
  Copy-Item -Path web/dist/* -Destination $target -Recurse
  $env:GOOS='linux'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
  foreach ($item in @(@('server','skilltree'),@('migrate-postgres','skilltree-migrate'),@('repair-migration','skilltree-repair'),@('backup','skilltree-backup'))) {
    go build -trimpath -o "bin/$($item[1])" "./cmd/$($item[0])"
    if ($LASTEXITCODE) { throw "build failed: $($item[0])" }
  }
} finally { Pop-Location }
