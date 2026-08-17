[CmdletBinding()]
param(
    [string]$ApiBase = "http://127.0.0.1:8080",
    [string]$AuthBase = "http://127.0.0.1:9099",
    [ValidateSet("emulator", "firebase")]
    [string]$AuthMode = "emulator",
    [string]$ApiKey = "rentstage-local-api-key",
    [string]$Email = "owner@rentstage.local",
    [string]$Password = "RentStage123!",
    [string]$PackageSlug = "paquete-fiesta-100-personas"
)

. "$PSScriptRoot/lib/smoke-common.ps1"

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

try {
    Write-Step "Checking API readiness"
    $ready = Invoke-RestMethod -Method Get -Uri "$ApiBase/readyz"
    if ($ready.status -ne "ready") {
        throw "API readiness returned an unexpected payload."
    }

    Write-Step "Signing in through Firebase Authentication"
    $signIn = Invoke-RentStageFirebasePasswordSignIn `
        -Mode $AuthMode `
        -AuthBase $AuthBase `
        -ApiKey $ApiKey `
        -Email $Email `
        -Password $Password

    if (-not $signIn.idToken) {
        throw "Firebase Authentication did not return an ID token."
    }

    $webSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession

    Write-Step "Requesting a CSRF token"
    $csrfResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $webSession

    if (-not $csrfResponse.csrf_token) {
        throw "The API did not return a CSRF token."
    }

    $headers = @{ "X-CSRF-Token" = $csrfResponse.csrf_token }

    Write-Step "Creating the authenticated RentStage session"
    $sessionBody = @{ id_token = $signIn.idToken } | ConvertTo-Json
    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $sessionBody | Out-Null

    Write-Step "Confirming package permissions in the active workspace"
    $me = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $webSession

    if (@($me.permissions) -notcontains "package.read") {
        throw "The authenticated role does not expose package.read."
    }

    Write-Step "Loading the active package catalog"
    $packageResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/packages?active=true" `
        -WebSession $webSession

    $package = @($packageResponse.items) | Where-Object { $_.slug -eq $PackageSlug } | Select-Object -First 1
    if (-not $package) {
        throw "Package '$PackageSlug' was not found. Confirm SEED_DEMO_DATA=true or create the package before running this smoke test."
    }
    if (-not $package.ready) {
        throw "Package '$PackageSlug' exists but is not ready. Review archived or missing resources."
    }

    Write-Step "Reading package composition"
    $detail = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/packages/$($package.id)" `
        -WebSession $webSession

    if (@($detail.items).Count -lt 1) {
        throw "The package contains no resources."
    }
    if ($PackageSlug -eq "paquete-fiesta-100-personas") {
        if (@($detail.items).Count -ne 5) {
            throw "The seeded Fiesta package should contain exactly 5 resources."
        }
        if ([int]$detail.total_quantity -ne 8) {
            throw "The seeded Fiesta package should contain 8 total units."
        }
    }

    Write-Step "Expanding two package units into quote snapshots"
    $template = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/packages/$($package.id)/quote-template?quantity=2" `
        -WebSession $webSession

    if ($template.package_quantity -ne 2) {
        throw "The quote template did not preserve the requested package quantity."
    }

    [decimal]$lineTotal = 0
    foreach ($line in @($template.items)) {
        $lineTotal += [decimal]$line.line_total
    }
    [decimal]$commercialTotal = [Math]::Round($lineTotal + [decimal]$template.extra_charges, 2)
    [decimal]$expectedTotal = [Math]::Round([decimal]$template.effective_price, 2)
    if ($commercialTotal -ne $expectedTotal) {
        throw "Expanded quote lines total $commercialTotal but the package commercial total is $expectedTotal."
    }

    Write-Step "Checking package availability in a future period"
    $start = (Get-Date).Date.AddDays(90).AddHours(14)
    $end = (Get-Date).Date.AddDays(90).AddHours(23)
    $availabilityBody = @{
        start_at = $start.ToUniversalTime().ToString("o")
        end_at = $end.ToUniversalTime().ToString("o")
        quantity = 1
    } | ConvertTo-Json

    $availability = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/packages/$($package.id)/availability" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $availabilityBody

    if ($availability.package_id -ne $package.id -or [int]$availability.package_quantity -ne 1) {
        throw "The availability response does not identify the requested package and quantity."
    }
    if (@($availability.items).Count -ne @($detail.items).Count) {
        throw "The availability response does not contain one result per package resource."
    }
    $conflicts = @($availability.items) | Where-Object { -not $_.can_fulfill } | ForEach-Object { $_.resource_name }
    $availabilityState = if ($availability.available) { "AVAILABLE" } else { "CONFLICT: $($conflicts -join ', ')" }

    Write-Host "Package:            $($detail.name)" -ForegroundColor Green
    Write-Host "Resources:          $(@($detail.items).Count)" -ForegroundColor Green
    Write-Host "Units per package:  $($detail.total_quantity)" -ForegroundColor Green
    Write-Host "Price per package:  $($detail.effective_price)" -ForegroundColor Green
    Write-Host "Two-package total:  $expectedTotal" -ForegroundColor Green
    Write-Host "Availability check:  $availabilityState" -ForegroundColor Green

    Write-Step "Closing the server session"
    Invoke-RestMethod `
        -Method Delete `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers | Out-Null

    Write-Host "`nRentStage package smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage Packages Core smoke test failed: $($_.Exception.Message)"
    exit 1
}
