[CmdletBinding()]
param(
    [switch]$SkipInstall
)

$ErrorActionPreference = 'Stop'
$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$previousCache = $env:GOCACHE

try {
    Set-Location -LiteralPath $projectRoot
    $env:GOCACHE = Join-Path $projectRoot '.cache\go-build'

    if (-not $SkipInstall) {
        & pnpm --dir frontend install --frozen-lockfile
        if ($LASTEXITCODE -ne 0) { throw 'frontend dependency installation failed' }
    }

    & go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed' }
    & go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go vet failed' }
    & pnpm --dir frontend test
    if ($LASTEXITCODE -ne 0) { throw 'frontend tests failed' }
    & pnpm --dir frontend run lint:api
    if ($LASTEXITCODE -ne 0) { throw 'OpenAPI validation failed' }
    & pnpm --dir frontend build
    if ($LASTEXITCODE -ne 0) { throw 'frontend build failed' }
    & (Join-Path $projectRoot 'scripts\check-lines.ps1')
    if ($LASTEXITCODE -ne 0) { throw 'source line limit check failed' }

    Write-Host 'All verification checks passed.'
}
finally {
    $env:GOCACHE = $previousCache
}
