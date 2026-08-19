$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$root = Resolve-Path (Join-Path $PSScriptRoot "../..")
Set-Location $root

$engineCommand = Get-Command pwsh -ErrorAction SilentlyContinue
if ($null -eq $engineCommand) {
    $engineCommand = Get-Command powershell -ErrorAction SilentlyContinue
}
if ($null -eq $engineCommand) {
    throw "Neither PowerShell 7 (pwsh) nor Windows PowerShell (powershell) is available in PATH."
}

$enginePath = $engineCommand.Source
if ([string]::IsNullOrWhiteSpace($enginePath)) {
    $enginePath = $engineCommand.Path
}
if ([string]::IsNullOrWhiteSpace($enginePath)) {
    $enginePath = $engineCommand.Name
}

Write-Host "PowerShell smoke-test engine: $enginePath" -ForegroundColor DarkGray

$tests = @(
    @{ File = ".\scripts\smoke-auth.ps1"; Arguments = @() },
    @{ File = ".\scripts\smoke-packages.ps1"; Arguments = @() },
    @{ File = ".\scripts\smoke-assistant.ps1"; Arguments = @() },
    @{ File = ".\scripts\smoke-public-catalog.ps1"; Arguments = @("-SkipSubmission") },
    @{ File = ".\scripts\smoke-quote-portal.ps1"; Arguments = @() },
    @{ File = ".\scripts\smoke-billing.ps1"; Arguments = @() },
    @{ File = ".\scripts\smoke-dte.ps1"; Arguments = @() }
)

foreach ($test in $tests) {
    Write-Host ""
    Write-Host "==> Running $($test.File)" -ForegroundColor Cyan

    $processArguments = @(
        "-NoLogo",
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $test.File
    ) + $test.Arguments

    & $enginePath @processArguments
    if ($LASTEXITCODE -ne 0) {
        throw "$($test.File) failed with exit code $LASTEXITCODE"
    }
}

Write-Host ""
Write-Host "All RentStage local integration smoke tests passed." -ForegroundColor Green
