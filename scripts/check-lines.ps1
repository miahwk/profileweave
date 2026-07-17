[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$projectRoot = [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$excludedDirectories = @(
    '.agents',
    '.cache',
    '.codex',
    '.git',
    '.pnpm-store',
    'dist',
    'generated',
    'node_modules',
    'vendor'
)

function Get-SourceFiles {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Root,

        [Parameter(Mandatory = $true)]
        [scriptblock]$Include
    )

    if (-not (Test-Path -LiteralPath $Root -PathType Container)) {
        return
    }

    $pending = New-Object 'System.Collections.Generic.Stack[string]'
    $pending.Push([System.IO.Path]::GetFullPath($Root))

    while ($pending.Count -gt 0) {
        $directory = $pending.Pop()

        foreach ($childDirectory in @(Get-ChildItem -LiteralPath $directory -Directory -Force -ErrorAction Stop)) {
            if ($excludedDirectories -notcontains $childDirectory.Name -and
                -not ($childDirectory.Attributes -band [System.IO.FileAttributes]::ReparsePoint)) {
                $pending.Push($childDirectory.FullName)
            }
        }

        foreach ($file in @(Get-ChildItem -LiteralPath $directory -File -Force -ErrorAction Stop)) {
            if (& $Include $file) {
                $file
            }
        }
    }
}

function Test-GeneratedSource {
    param(
        [Parameter(Mandatory = $true)]
        [System.IO.FileInfo]$File
    )

    if ($File.Name -match '\.(?:gen|generated)\.(?:go|ts)$') {
        return $true
    }

    if ($File.Extension -ne '.go') {
        return $false
    }

    $stream = $null
    $reader = $null
    try {
        $stream = New-Object System.IO.FileStream(
            $File.FullName,
            [System.IO.FileMode]::Open,
            [System.IO.FileAccess]::Read,
            ([System.IO.FileShare]::ReadWrite -bor [System.IO.FileShare]::Delete)
        )
        $reader = New-Object System.IO.StreamReader($stream, $true)
        for ($lineNumber = 0; $lineNumber -lt 20 -and -not $reader.EndOfStream; $lineNumber++) {
            $line = $reader.ReadLine()
            if ($line -match '^// Code generated .* DO NOT EDIT\.$') {
                return $true
            }
        }
        return $false
    }
    finally {
        if ($null -ne $reader) {
            $reader.Dispose()
        }
        elseif ($null -ne $stream) {
            $stream.Dispose()
        }
    }
}

function Get-LineCount {
    param(
        [Parameter(Mandatory = $true)]
        [System.IO.FileInfo]$File
    )

    $stream = $null
    $reader = $null
    try {
        $stream = New-Object System.IO.FileStream(
            $File.FullName,
            [System.IO.FileMode]::Open,
            [System.IO.FileAccess]::Read,
            ([System.IO.FileShare]::ReadWrite -bor [System.IO.FileShare]::Delete)
        )
        $reader = New-Object System.IO.StreamReader($stream, $true)
        $count = 0
        while ($null -ne $reader.ReadLine()) {
            $count++
        }
        return $count
    }
    finally {
        if ($null -ne $reader) {
            $reader.Dispose()
        }
        elseif ($null -ne $stream) {
            $stream.Dispose()
        }
    }
}

function Get-RelativePath {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $fullPath = [System.IO.Path]::GetFullPath($Path)
    $rootWithSeparator = $projectRoot.TrimEnd('\', '/') + [System.IO.Path]::DirectorySeparatorChar
    if ($fullPath.StartsWith($rootWithSeparator, [System.StringComparison]::OrdinalIgnoreCase)) {
        return $fullPath.Substring($rootWithSeparator.Length)
    }
    return $fullPath
}

$rules = @(
    [pscustomobject]@{
        Name = 'Frontend Vue/TypeScript'
        Root = Join-Path $projectRoot 'frontend\src'
        Limit = 400
        Include = {
            param($file)
            $file.Extension -in @('.ts', '.vue') -and -not (Test-GeneratedSource -File $file)
        }
    },
    [pscustomobject]@{
        Name = 'Go'
        Root = $projectRoot
        Limit = 300
        Include = {
            param($file)
            $file.Extension -eq '.go' -and -not (Test-GeneratedSource -File $file)
        }
    }
)

$checkedCount = 0
$violations = New-Object System.Collections.Generic.List[object]

foreach ($rule in $rules) {
    $ruleCount = 0
    foreach ($file in @(Get-SourceFiles -Root $rule.Root -Include $rule.Include)) {
        $lineCount = Get-LineCount -File $file
        $ruleCount++
        $checkedCount++
        if ($lineCount -gt $rule.Limit) {
            $violations.Add([pscustomobject]@{
                Rule = $rule.Name
                Path = Get-RelativePath -Path $file.FullName
                Lines = $lineCount
                Limit = $rule.Limit
            })
        }
    }
    Write-Host ("{0}: checked {1} file(s), limit {2} lines." -f $rule.Name, $ruleCount, $rule.Limit)
}

if ($violations.Count -gt 0) {
    Write-Host ''
    Write-Error ("Line limit check failed: {0} file(s) exceed their limit." -f $violations.Count) -ErrorAction Continue
    foreach ($violation in @($violations | Sort-Object Path)) {
        Write-Host ("  {0}: {1} lines (limit {2}, {3} over) [{4}]" -f `
            $violation.Path,
            $violation.Lines,
            $violation.Limit,
            ($violation.Lines - $violation.Limit),
            $violation.Rule)
    }
    exit 1
}

Write-Host ("Line limit check passed: {0} source file(s) checked." -f $checkedCount)
