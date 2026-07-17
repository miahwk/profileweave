[CmdletBinding()]
param(
    [switch]$SkipVerify,
    [string]$Version = 'dev',
    [string]$Commit = 'unknown',
    [string]$BuildDate = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
)

$ErrorActionPreference = 'Stop'
$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$outputDir = Join-Path $projectRoot 'dist'
$previousCache = $env:GOCACHE

try {
    Set-Location -LiteralPath $projectRoot
    if (-not $SkipVerify) {
        & (Join-Path $projectRoot 'scripts\verify.ps1')
        if ($LASTEXITCODE -ne 0) { throw 'verification failed' }
    }
    $expectedOutput = [System.IO.Path]::GetFullPath((Join-Path $projectRoot 'dist'))
    if ([System.IO.Path]::GetFullPath($outputDir) -ne $expectedOutput) {
        throw 'refusing to clean an unexpected output directory'
    }
    if (Test-Path -LiteralPath $outputDir) {
        Remove-Item -LiteralPath $outputDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
    $env:GOCACHE = Join-Path $projectRoot '.cache\go-build'
    $linkerFlags = "-s -w -X github.com/miahwk/profileweave/internal/buildinfo.Version=$Version -X github.com/miahwk/profileweave/internal/buildinfo.Commit=$Commit -X github.com/miahwk/profileweave/internal/buildinfo.Date=$BuildDate"
    & go build -buildvcs=false -trimpath -ldflags $linkerFlags -o (Join-Path $outputDir 'profileweave.exe') ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw 'Go build failed' }
    $frontendBuild = Join-Path $projectRoot 'frontend\dist'
    if (-not (Test-Path -LiteralPath (Join-Path $frontendBuild 'index.html') -PathType Leaf)) {
        throw 'frontend build is missing; run pnpm --dir frontend build first'
    }
    $webOutput = Join-Path $outputDir 'frontend\dist'
    New-Item -ItemType Directory -Force -Path $webOutput | Out-Null
    Copy-Item -Path (Join-Path $frontendBuild '*') -Destination $webOutput -Recurse -Force
    foreach ($artifact in @('README.md', 'LICENSE', 'NOTICE', 'THIRD_PARTY_NOTICES.md', 'CHANGELOG.md')) {
        $source = Join-Path $projectRoot $artifact
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            throw "required distribution file is missing: $artifact"
        }
        Copy-Item -LiteralPath $source -Destination $outputDir -Force
    }
    Write-Host "Built $outputDir\profileweave.exe ($Version, $Commit)"
}
finally {
    $env:GOCACHE = $previousCache
}
