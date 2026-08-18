[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$WebBase,

    [Parameter(Mandatory = $true)]
    [string]$ApiServiceUrl,

    [Parameter(Mandatory = $true)]
    [string]$ApiKey,

    [Parameter(Mandatory = $true)]
    [string]$Email,

    [Parameter(Mandatory = $true)]
    [string]$Password,

    [string]$TenantSlug = "audiopro-demo",
    [string]$PackageSlug = "paquete-fiesta-100-personas",
    [switch]$ReadOnly
)

$ErrorActionPreference = "Stop"
$web = $WebBase.TrimEnd("/")
$privateApi = $ApiServiceUrl.TrimEnd("/")
if (-not $web.StartsWith("https://", [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Staging smoke tests require an HTTPS WebBase."
}
if (-not $privateApi.StartsWith("https://", [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "Staging smoke tests require an HTTPS ApiServiceUrl."
}
$apiBase = "$web/api/backend"
$engine = (Get-Process -Id $PID).Path

function Assert-StatusCode {
    param([string]$Uri, [int]$Expected)
    try {
        $response = Invoke-WebRequest -Uri $Uri -UseBasicParsing
        $actual = [int]$response.StatusCode
    }
    catch {
        if ($_.Exception.Response) {
            $actual = [int]$_.Exception.Response.StatusCode
        }
        else {
            throw
        }
    }
    if ($actual -ne $Expected) {
        throw "$Uri returned HTTP $actual; expected $Expected."
    }
}

function Invoke-SmokeScript {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    Write-Host "`n============================================================" -ForegroundColor DarkGray
    Write-Host "Running $Name against $web" -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor DarkGray

    & $engine -NoLogo -NoProfile -ExecutionPolicy Bypass -File "$PSScriptRoot/$Name" @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE."
    }
}

Write-Host "==> Checking the public web service" -ForegroundColor Cyan
$webHealth = Invoke-RestMethod -Uri "$web/api/healthz"
if ($webHealth.status -ne "ok") {
    throw "The web health endpoint returned an unexpected payload."
}
Assert-StatusCode -Uri "$web/login" -Expected 200

Write-Host "==> Checking the private API through the authenticated server proxy" -ForegroundColor Cyan
$apiReady = Invoke-RestMethod -Uri "$apiBase/readyz"
if ($apiReady.status -ne "ready") {
    throw "The proxied API readiness check returned an unexpected payload."
}

Write-Host "==> Confirming that the API Cloud Run service is not anonymous" -ForegroundColor Cyan
try {
    $direct = Invoke-WebRequest -Uri "$privateApi/healthz" -UseBasicParsing
    $directStatus = [int]$direct.StatusCode
}
catch {
    if (-not $_.Exception.Response) { throw }
    $directStatus = [int]$_.Exception.Response.StatusCode
}
# Depending on the Cloud Run URL and authentication boundary, an anonymous
# request can be rejected as Unauthorized, Forbidden, or concealed as Not Found.
if ($directStatus -notin @(401, 403, 404)) {
    throw "Direct anonymous API access returned HTTP $directStatus; expected 401, 403, or 404."
}

Write-Host "==> Checking Quote Portal response protections" -ForegroundColor Cyan
$q = Invoke-WebRequest -Uri "$web/q" -UseBasicParsing
if ([string]$q.Headers["Cache-Control"] -notmatch "no-store") {
    throw "The Quote Portal response is missing Cache-Control: no-store."
}
if ([string]$q.Headers["Referrer-Policy"] -ne "no-referrer") {
    throw "The Quote Portal response is missing Referrer-Policy: no-referrer."
}

$common = @(
    "-ApiBase", $apiBase,
    "-AuthMode", "firebase",
    "-AuthBase", "https://identitytoolkit.googleapis.com",
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
    Invoke-SmokeScript -Name "smoke-quote-portal.ps1" -Arguments $common
    Invoke-SmokeScript -Name "smoke-billing.ps1" -Arguments $common
    Invoke-SmokeScript -Name "smoke-dte.ps1" -Arguments $common
}

Write-Host "`nRentStage staging smoke suite passed." -ForegroundColor Green
