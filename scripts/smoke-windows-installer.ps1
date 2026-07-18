[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$InstallerPath,
    [Parameter(Mandatory)]
    [string]$Version
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Net.Http
$installer = [System.IO.Path]::GetFullPath($InstallerPath)
if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) {
    throw "installer not found: $installer"
}
if ($Version -notmatch '^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
    throw "invalid expected version: $Version"
}
$expectedVersion = $Version.TrimStart('v')
$appId = '{E6784F49-0EE8-4D07-A8D9-E9A64B1D53B8}_is1'
$uninstallRegistryPath = "Registry::HKEY_CURRENT_USER\Software\Microsoft\Windows\CurrentVersion\Uninstall\$appId"
$defaultInstall = Join-Path $env:LOCALAPPDATA 'Programs\ProfileWeave\profileweave.exe'
if ((Test-Path -LiteralPath $uninstallRegistryPath) -or (Test-Path -LiteralPath $defaultInstall)) {
    throw 'refusing installer smoke test because ProfileWeave is already installed for this user'
}

$temporaryRoot = [System.IO.Path]::GetFullPath([System.IO.Path]::GetTempPath()).TrimEnd('\', '/')
$testRoot = [System.IO.Path]::GetFullPath((Join-Path $temporaryRoot ("profileweave-installer-smoke-" + [guid]::NewGuid().ToString('N'))))
$testPrefix = $temporaryRoot + [System.IO.Path]::DirectorySeparatorChar
if (-not $testRoot.StartsWith($testPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
    throw 'refusing to create installer smoke directory outside the system temporary directory'
}
$installDirectory = Join-Path $testRoot 'app'
$dataDirectory = Join-Path $testRoot 'data'
$sentinelPath = Join-Path $dataDirectory 'retain-after-uninstall.sentinel'
$logPath = Join-Path $dataDirectory 'logs\smoke.log'
$programGroup = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::Programs)) 'ProfileWeave'
$launchShortcut = Join-Path $programGroup 'ProfileWeave.lnk'
$exitShortcut = Join-Path $programGroup 'Exit ProfileWeave.lnk'
if (Test-Path -LiteralPath $programGroup) {
    throw "refusing installer smoke test because the Start Menu group already exists: $programGroup"
}
$serverProcess = $null
$previousDataDirectory = $env:PROFILEWEAVE_DATA_DIR
$previousPort = $env:PROFILEWEAVE_PORT

function Invoke-AndWait {
    param([string]$FilePath, [string[]]$Arguments)
    $process = Start-Process -FilePath $FilePath -ArgumentList $Arguments -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "$FilePath failed with exit code $($process.ExitCode)"
    }
}

try {
    New-Item -ItemType Directory -Force -Path $dataDirectory | Out-Null
    Set-Content -LiteralPath $sentinelPath -Value 'ProfileWeave installer retention smoke test' -Encoding utf8
    $portProbe = [System.Net.Sockets.TcpListener]::new([System.Net.IPAddress]::Loopback, 0)
    $portProbe.Start()
    $port = ([System.Net.IPEndPoint]$portProbe.LocalEndpoint).Port
    $portProbe.Stop()
    $env:PROFILEWEAVE_DATA_DIR = $dataDirectory
    $env:PROFILEWEAVE_PORT = $port.ToString()

    Invoke-AndWait -FilePath $installer -Arguments @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', "/DIR=`"$installDirectory`""
    )

    $executable = Join-Path $installDirectory 'profileweave.exe'
    $uninstaller = Join-Path $installDirectory 'unins000.exe'
    if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {
        throw 'installed executable is missing'
    }
    if (-not (Test-Path -LiteralPath (Join-Path $installDirectory 'frontend\dist\index.html') -PathType Leaf)) {
        throw 'installed frontend bundle is missing'
    }
    if (-not (Test-Path -LiteralPath $uninstaller -PathType Leaf)) {
        throw 'uninstaller is missing'
    }
    if (-not (Test-Path -LiteralPath $launchShortcut -PathType Leaf)) {
        throw 'Start Menu launch shortcut is missing'
    }
    if (-not (Test-Path -LiteralPath $exitShortcut -PathType Leaf)) {
        throw 'Start Menu exit shortcut is missing'
    }

    # Reinstall the same AppId to exercise the in-place upgrade path before launch.
    Invoke-AndWait -FilePath $installer -Arguments @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', "/DIR=`"$installDirectory`""
    )
    if (-not (Test-Path -LiteralPath $sentinelPath -PathType Leaf)) {
        throw 'in-place upgrade removed user data'
    }
    Invoke-AndWait -FilePath $executable -Arguments @('--version')

    $serverProcess = Start-Process -FilePath $executable -ArgumentList @('--log-file', "`"$logPath`"") -PassThru

    $handler = [System.Net.Http.HttpClientHandler]::new()
    $handler.UseProxy = $false
    $client = [System.Net.Http.HttpClient]::new($handler)
    try {
        $health = $null
        $deadline = [DateTime]::UtcNow.AddSeconds(20)
        while ([DateTime]::UtcNow -lt $deadline) {
            if ($serverProcess.HasExited) {
                throw "installed server exited during startup with code $($serverProcess.ExitCode)"
            }
            try {
                $json = $client.GetStringAsync("http://127.0.0.1:$port/api/v1/health").GetAwaiter().GetResult()
                $health = $json | ConvertFrom-Json
                break
            } catch {
                Start-Sleep -Milliseconds 250
            }
        }
        if ($null -eq $health) { throw 'installed server did not become healthy within 20 seconds' }
        if ($health.product -ne 'ProfileWeave') { throw "unexpected health product identity: $($health.product)" }
        if ($health.version -ne $expectedVersion) { throw "unexpected installed version: $($health.version)" }
    }
    finally {
        $client.Dispose()
        $handler.Dispose()
    }

    Invoke-AndWait -FilePath $executable -Arguments @('--shutdown')
    if (-not $serverProcess.WaitForExit(15000)) {
        throw 'installed server did not exit after the shutdown command'
    }

    Invoke-AndWait -FilePath $uninstaller -Arguments @('/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART')
    $deadline = [DateTime]::UtcNow.AddSeconds(15)
    while ((Test-Path -LiteralPath $installDirectory) -and [DateTime]::UtcNow -lt $deadline) {
        Start-Sleep -Milliseconds 250
    }
    if (Test-Path -LiteralPath $installDirectory) {
        throw 'silent uninstall left the application directory behind'
    }
    if ((Test-Path -LiteralPath $launchShortcut) -or (Test-Path -LiteralPath $exitShortcut)) {
        throw 'silent uninstall left Start Menu shortcuts behind'
    }
    if (-not (Test-Path -LiteralPath $sentinelPath -PathType Leaf)) {
        throw 'silent uninstall removed user data'
    }
    Write-Host "Windows installer smoke test passed for ProfileWeave $expectedVersion"
}
finally {
    $cleanupSucceeded = $true
    if ($null -ne $serverProcess -and -not $serverProcess.HasExited) {
        Stop-Process -Id $serverProcess.Id -Force -ErrorAction SilentlyContinue
        $serverProcess.WaitForExit(5000) | Out-Null
    }
    $cleanupUninstaller = Join-Path $installDirectory 'unins000.exe'
    if (Test-Path -LiteralPath $cleanupUninstaller -PathType Leaf) {
        try {
            $cleanup = Start-Process -FilePath $cleanupUninstaller -ArgumentList @(
                '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART'
            ) -Wait -PassThru
            if ($cleanup.ExitCode -ne 0) {
                $cleanupSucceeded = $false
                Write-Warning "cleanup uninstaller exited with code $($cleanup.ExitCode)"
            }
        } catch {
            $cleanupSucceeded = $false
            Write-Warning "cleanup uninstall failed: $($_.Exception.Message)"
        }
    }
    if ((Test-Path -LiteralPath $uninstallRegistryPath) -or (Test-Path -LiteralPath $programGroup)) {
        $cleanupSucceeded = $false
        Write-Warning "cleanup left installer registration or shortcuts; recovery files are preserved at $testRoot"
    }
    if ($null -eq $previousDataDirectory) {
        Remove-Item Env:PROFILEWEAVE_DATA_DIR -ErrorAction SilentlyContinue
    } else {
        $env:PROFILEWEAVE_DATA_DIR = $previousDataDirectory
    }
    if ($null -eq $previousPort) {
        Remove-Item Env:PROFILEWEAVE_PORT -ErrorAction SilentlyContinue
    } else {
        $env:PROFILEWEAVE_PORT = $previousPort
    }
    if ($cleanupSucceeded -and (Test-Path -LiteralPath $testRoot)) {
        Remove-Item -LiteralPath $testRoot -Recurse -Force
    }
}
