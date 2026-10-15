#!/usr/bin/env pwsh
#MISE description="Build and run a temporary native Go binary"
#MISE hide=true
#MISE raw_args=true
#MISE quiet=true
#MISE raw=true
#MISE dir="{{config_root}}"

$ErrorActionPreference = 'Stop'
$status = 6
$temporary = $null
$process = $null
$originalLocation = Get-Location
$originalGOOS = $env:GOOS
$originalGOARCH = $env:GOARCH
$originalGOWORK = $env:GOWORK

function Start-NativeProgram([string] $executable, [string[]] $arguments, [string] $directory) {
    $script:process = [System.Diagnostics.Process]::new()
    $script:process.StartInfo = [System.Diagnostics.ProcessStartInfo]::new()
    $script:process.StartInfo.FileName = $executable
    $script:process.StartInfo.WorkingDirectory = $directory
    $script:process.StartInfo.UseShellExecute = $false
    foreach ($argument in $arguments) {
        $script:process.StartInfo.ArgumentList.Add($argument)
    }
    if (-not $script:process.Start()) {
        throw 'Could not start native program.'
    }
    while (-not $script:process.WaitForExit(100)) {}
    $exitCode = $script:process.ExitCode
    $script:process.Dispose()
    $script:process = $null
    return $exitCode
}

try {
    if ($args.Count -lt 2) {
        throw 'usage: go:run <existing-working-directory> <package> [arguments...]'
    }
    $workingDirectory = [string] $args[0]
    $package = [string] $args[1]
    if (-not (Test-Path -LiteralPath $workingDirectory -PathType Container)) {
        throw "Working directory does not exist: $workingDirectory"
    }
    $workingDirectory = (Resolve-Path -LiteralPath $workingDirectory).Path
    Set-Location -LiteralPath $workingDirectory
    $programArguments = [string[]] @()
    if ($args.Count -gt 2) {
        $programArguments = [string[]] $args[2..($args.Count - 1)]
    }
    $go = (Get-Command go -CommandType Application).Source
    $env:GOWORK = 'off'
    $hostTarget = @(& $go env GOHOSTOS GOHOSTARCH)
    if ($LASTEXITCODE -ne 0 -or $hostTarget.Count -ne 2) {
        throw 'Could not determine the native Go target.'
    }
    $env:GOOS = $hostTarget[0]
    $env:GOARCH = $hostTarget[1]
    $temporary = Join-Path ([System.IO.Path]::GetTempPath()) ('cli4g-go-run.' + [guid]::NewGuid().ToString('N'))
    [void] [System.IO.Directory]::CreateDirectory($temporary)
    $binary = Join-Path $temporary 'program.exe'
    $buildCode = Start-NativeProgram $go @('build', '-o', $binary, $package) $workingDirectory
    if ($buildCode -eq 0) {
        $status = Start-NativeProgram $binary $programArguments $workingDirectory
    }
} catch {
    [Console]::Error.WriteLine($_.Exception.Message)
    $status = 6
} finally {
    if ($null -ne $process) {
        try {
            if (-not $process.HasExited) {
                $process.Kill($true)
                [void] $process.WaitForExit(2000)
            }
        } catch {
            [Console]::Error.WriteLine($_.Exception.Message)
        }
        $process.Dispose()
    }
    if ($null -ne $temporary) {
        try {
            Remove-Item -LiteralPath $temporary -Recurse -Force
        } catch {
            [Console]::Error.WriteLine("Temporary binary cleanup failed: " + $_.Exception.Message)
        }
    }
    $env:GOOS = $originalGOOS
    $env:GOARCH = $originalGOARCH
    $env:GOWORK = $originalGOWORK
    Set-Location -LiteralPath $originalLocation.Path
}
exit $status
