[CmdletBinding()]
param(
    [string]$ApiBase = "http://127.0.0.1:8080",
    [string]$AuthBase = "http://127.0.0.1:9099",
    [ValidateSet("emulator", "firebase")]
    [string]$AuthMode = "emulator",
    [string]$ApiKey = "rentstage-local-api-key",
    [string]$Email = "owner@rentstage.local",
    [string]$Password = "RentStage123!",
    [string]$TenantSlug = "audiopro-demo",
    [string]$PackageSlug = "paquete-fiesta-100-personas",
    [switch]$ReadOnly
)

$ErrorActionPreference = "Stop"
$engine = (Get-Process -Id $PID).Path

function Invoke-SmokeScript {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    Write-Host "`n============================================================" -ForegroundColor DarkGray
    Write-Host "Running $Name" -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor DarkGray

    & $engine -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$PSScriptRoot/$Name" @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE."
    }
}

$common = @(
    "-ApiBase", $ApiBase,
    "-AuthMode", $AuthMode,
    "-AuthBase", $AuthBase,
    "-ApiKey", $ApiKey,
    "-Email", $Email,
    "-Password", $Password
)

Invoke-SmokeScript -Name "smoke-auth.ps1" -Arguments $common
Invoke-SmokeScript -Name "smoke-packages.ps1" -Arguments ($common + @("-PackageSlug", $PackageSlug))

$publicArgs = $common + @("-TenantSlug", $TenantSlug, "-PackageSlug", $PackageSlug)
if ($ReadOnly) {
    $publicArgs += "-SkipSubmission"
}
Invoke-SmokeScript -Name "smoke-public-catalog.ps1" -Arguments $publicArgs

if (-not $ReadOnly) {
    Invoke-SmokeScript -Name "smoke-assistant.ps1" -Arguments $common
    Invoke-SmokeScript -Name "smoke-quote-portal.ps1" -Arguments $common
    Invoke-SmokeScript -Name "smoke-billing.ps1" -Arguments $common
    Invoke-SmokeScript -Name "smoke-dte.ps1" -Arguments $common
}

Write-Host "`nRentStage smoke suite passed." -ForegroundColor Green
