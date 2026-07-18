[CmdletBinding()]
param(
    [ValidateSet('amd64', 'arm64')]
    [string]$Architecture = 'amd64',
    [string]$Version = '0.0.0-dev',
    [string]$Commit = 'unknown',
    [string]$BuildDate = (Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ'),
    [string]$OutputDirectory,
    [string]$InnoSetupVersion = '6.7.3',
    [string]$InnoSetupURL = 'https://github.com/jrsoftware/issrc/releases/download/is-6_7_3/innosetup-6.7.3.exe',
    [string]$InnoSetupSHA256 = '9c73c3bae7ed48d44112a0f48e66742c00090bdb5bef71d9d3c056c66e97b732',
    [string]$InnoSetupCompilerSHA256 = '0a8757031b33777e4c9cbffee40f11a5062b36d25cbe144c1db73b6102b80ad7',
    [string]$InnoSetupCompilerPath
)

$ErrorActionPreference = 'Stop'
$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$stagingRoot = [System.IO.Path]::GetFullPath((Join-Path $projectRoot 'dist\windows-installer'))
$architectureRoot = [System.IO.Path]::GetFullPath((Join-Path $stagingRoot $Architecture))
$payloadDirectory = Join-Path $architectureRoot 'payload'
$compilerRoot = [System.IO.Path]::GetFullPath((Join-Path $projectRoot ".cache\inno-setup\$InnoSetupVersion"))
$downloadRoot = [System.IO.Path]::GetFullPath((Join-Path $projectRoot '.cache\downloads'))
$frontendBuild = Join-Path $projectRoot 'frontend\dist'
$installerScript = Join-Path $projectRoot 'packaging\windows\profileweave.iss'

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $projectRoot 'release'
}
$OutputDirectory = [System.IO.Path]::GetFullPath($OutputDirectory)

function Assert-DescendantPath {
    param([string]$Candidate, [string]$Parent, [string]$Purpose)
    $resolvedCandidate = [System.IO.Path]::GetFullPath($Candidate).TrimEnd('\', '/')
    $resolvedParent = [System.IO.Path]::GetFullPath($Parent).TrimEnd('\', '/')
    $prefix = $resolvedParent + [System.IO.Path]::DirectorySeparatorChar
    if (-not $resolvedCandidate.StartsWith($prefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "refusing $Purpose outside $resolvedParent"
    }
}

function Assert-BuildValue {
    param([string]$Name, [string]$Value, [string]$Pattern)
    if ([string]::IsNullOrWhiteSpace($Value) -or $Value -notmatch $Pattern) {
        throw "invalid $Name value: $Value"
    }
}

function Restore-EnvironmentValue {
    param([string]$Name, [AllowNull()][string]$Value)
    if ($null -eq $Value) {
        Remove-Item -Path "Env:$Name" -ErrorAction SilentlyContinue
    } else {
        Set-Item -Path "Env:$Name" -Value $Value
    }
}

Assert-DescendantPath -Candidate $architectureRoot -Parent $stagingRoot -Purpose 'staging cleanup'
Assert-DescendantPath -Candidate $compilerRoot -Parent (Join-Path $projectRoot '.cache\inno-setup') -Purpose 'compiler cache use'
Assert-DescendantPath -Candidate $downloadRoot -Parent (Join-Path $projectRoot '.cache') -Purpose 'download cache use'
Assert-BuildValue -Name 'version' -Value $Version -Pattern '^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$'
Assert-BuildValue -Name 'commit' -Value $Commit -Pattern '^[0-9A-Za-z._-]+$'
Assert-BuildValue -Name 'build date' -Value $BuildDate -Pattern '^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$'
Assert-BuildValue -Name 'Inno Setup version' -Value $InnoSetupVersion -Pattern '^[0-9]+\.[0-9]+\.[0-9]+$'
Assert-BuildValue -Name 'Inno Setup SHA-256' -Value $InnoSetupSHA256 -Pattern '^[0-9a-fA-F]{64}$'
Assert-BuildValue -Name 'Inno Setup compiler SHA-256' -Value $InnoSetupCompilerSHA256 -Pattern '^[0-9a-fA-F]{64}$'

$releaseVersion = $Version.TrimStart('v')
$versionLabel = "v$releaseVersion"
$outputBaseFilename = "profileweave-$versionLabel-windows-$Architecture-setup"
$expectedInstaller = Join-Path $OutputDirectory "$outputBaseFilename.exe"

$previousEnvironment = @{
    GOOS = $env:GOOS
    GOARCH = $env:GOARCH
    CGO_ENABLED = $env:CGO_ENABLED
    GOCACHE = $env:GOCACHE
}

try {
    Set-Location -LiteralPath $projectRoot
    if (-not (Test-Path -LiteralPath (Join-Path $frontendBuild 'index.html') -PathType Leaf)) {
        throw 'frontend build is missing; run pnpm --dir frontend build first'
    }
    if (-not (Test-Path -LiteralPath $installerScript -PathType Leaf)) {
        throw "installer script is missing: $installerScript"
    }

    if (Test-Path -LiteralPath $architectureRoot) {
        Remove-Item -LiteralPath $architectureRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $payloadDirectory | Out-Null
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

    $env:GOOS = 'windows'
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = '0'
    $env:GOCACHE = Join-Path $projectRoot '.cache\go-build'
    $linkerFlags = "-H=windowsgui -s -w -X github.com/miahwk/profileweave/internal/buildinfo.Version=$releaseVersion -X github.com/miahwk/profileweave/internal/buildinfo.Commit=$Commit -X github.com/miahwk/profileweave/internal/buildinfo.Date=$BuildDate"
    & go build -buildvcs=true -trimpath -ldflags $linkerFlags -o (Join-Path $payloadDirectory 'profileweave.exe') ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw 'Windows GUI payload build failed' }

    $webOutput = Join-Path $payloadDirectory 'frontend\dist'
    New-Item -ItemType Directory -Force -Path $webOutput | Out-Null
    Copy-Item -Path (Join-Path $frontendBuild '*') -Destination $webOutput -Recurse -Force
    foreach ($artifact in @('README.md', 'LICENSE', 'NOTICE', 'THIRD_PARTY_NOTICES.md', 'CHANGELOG.md', 'SECURITY.md')) {
        $source = Join-Path $projectRoot $artifact
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            throw "required distribution file is missing: $artifact"
        }
        Copy-Item -LiteralPath $source -Destination $payloadDirectory -Force
    }

    $compiler = $InnoSetupCompilerPath
    if ([string]::IsNullOrWhiteSpace($compiler)) {
        $compiler = Join-Path $compilerRoot 'ISCC.exe'
        New-Item -ItemType Directory -Force -Path $downloadRoot | Out-Null
        New-Item -ItemType Directory -Force -Path $compilerRoot | Out-Null
        $compilerInstaller = Join-Path $downloadRoot "innosetup-$InnoSetupVersion.exe"
        Assert-DescendantPath -Candidate $compilerInstaller -Parent $downloadRoot -Purpose 'compiler download cleanup'
        if (-not (Test-Path -LiteralPath $compilerInstaller -PathType Leaf)) {
            $partialDownload = "$compilerInstaller.partial"
            Assert-DescendantPath -Candidate $partialDownload -Parent $downloadRoot -Purpose 'partial download cleanup'
            if (Test-Path -LiteralPath $partialDownload) {
                Remove-Item -LiteralPath $partialDownload -Force
            }
            Invoke-WebRequest -Uri $InnoSetupURL -OutFile $partialDownload
            Move-Item -LiteralPath $partialDownload -Destination $compilerInstaller -Force
        }
        $actualHash = (Get-FileHash -LiteralPath $compilerInstaller -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actualHash -ne $InnoSetupSHA256.ToLowerInvariant()) {
            Remove-Item -LiteralPath $compilerInstaller -Force
            throw "Inno Setup download SHA-256 mismatch: expected $InnoSetupSHA256, got $actualHash"
        }
        if (-not (Test-Path -LiteralPath $compiler -PathType Leaf)) {
            $compilerInstallLog = Join-Path $compilerRoot 'install.log'
            Assert-DescendantPath -Candidate $compilerInstallLog -Parent $compilerRoot -Purpose 'compiler install log use'
            $compilerInstall = Start-Process -FilePath $compilerInstaller -ArgumentList @(
                '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', '/CURRENTUSER',
                "/DIR=`"$compilerRoot`"", "/LOG=`"$compilerInstallLog`""
            ) -Wait -PassThru
            if ($compilerInstall.ExitCode -ne 0) {
                throw "Inno Setup compiler installation failed with exit code $($compilerInstall.ExitCode)"
            }
        }
    }
    $compiler = [System.IO.Path]::GetFullPath($compiler)
    if (-not (Test-Path -LiteralPath $compiler -PathType Leaf)) {
        throw "Inno Setup compiler not found: $compiler"
    }
    $actualCompilerHash = (Get-FileHash -LiteralPath $compiler -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualCompilerHash -ne $InnoSetupCompilerSHA256.ToLowerInvariant()) {
        throw "Inno Setup compiler SHA-256 mismatch: expected $InnoSetupCompilerSHA256, got $actualCompilerHash"
    }

    $targetDefine = if ($Architecture -eq 'amd64') { '/DTargetAMD64' } else { '/DTargetARM64' }
    & $compiler "/DAppVersion=$releaseVersion" "/DPayloadDir=$payloadDirectory" "/DOutputDir=$OutputDirectory" "/DOutputBaseFilename=$outputBaseFilename" $targetDefine $installerScript | Out-Host
    $compilerExitCode = $LASTEXITCODE
    if ($compilerExitCode -ne 0) { throw 'Inno Setup compilation failed' }
    if (-not (Test-Path -LiteralPath $expectedInstaller -PathType Leaf)) {
        throw "Inno Setup did not produce the expected installer: $expectedInstaller"
    }
    Write-Output $expectedInstaller
}
finally {
    Restore-EnvironmentValue -Name GOOS -Value $previousEnvironment.GOOS
    Restore-EnvironmentValue -Name GOARCH -Value $previousEnvironment.GOARCH
    Restore-EnvironmentValue -Name CGO_ENABLED -Value $previousEnvironment.CGO_ENABLED
    Restore-EnvironmentValue -Name GOCACHE -Value $previousEnvironment.GOCACHE
}
