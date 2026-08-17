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
    [switch]$SkipSubmission
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

    Write-Step "Loading the anonymous tenant catalog"
    $catalog = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/public/catalogs/$TenantSlug"

    if ($catalog.tenant.slug -ne $TenantSlug) {
        throw "The public catalog resolved a different tenant slug."
    }
    if (-not $catalog.settings.quote_requests_enabled) {
        throw "The public catalog is not accepting quote requests. Enable them before running this smoke test."
    }

    $package = @($catalog.packages) | Where-Object { $_.slug -eq $PackageSlug } | Select-Object -First 1
    if (-not $package) {
        throw "Public package '$PackageSlug' was not found. Publish it before running this smoke test."
    }

    Write-Step "Reading the public package detail"
    $packageResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/public/catalogs/$TenantSlug/packages/$PackageSlug"

    if ($packageResponse.package.slug -ne $PackageSlug -or @($packageResponse.package.items).Count -lt 1) {
        throw "The public package detail is incomplete."
    }

    $webSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession

    Write-Step "Requesting anonymous CSRF protection"
    $csrfResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $webSession

    if (-not $csrfResponse.csrf_token) {
        throw "The API did not return a CSRF token."
    }

    $headers = @{ "X-CSRF-Token" = $csrfResponse.csrf_token }
    $start = (Get-Date).Date.AddDays(120).AddHours(14)
    $end = $start.AddHours(9)
    $selection = @(@{ package_slug = $PackageSlug; quantity = 1 })

    Write-Step "Checking public package availability"
    $availabilityBody = @{
        start_at = $start.ToUniversalTime().ToString("o")
        end_at = $end.ToUniversalTime().ToString("o")
        selections = $selection
    } | ConvertTo-Json -Depth 8

    $availability = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/public/catalogs/$TenantSlug/availability" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $availabilityBody

    if (@($availability.items).Count -lt 1) {
        throw "Public availability returned no resource results."
    }
    foreach ($item in @($availability.items)) {
        if ($null -ne $item.resource_id -or $null -ne $item.available_quantity -or $null -ne $item.eligible_assets) {
            throw "Public availability exposed internal inventory identifiers or capacity details."
        }
    }

    if ($SkipSubmission) {
        Write-Host "Tenant:              $($catalog.tenant.name)" -ForegroundColor Green
        Write-Host "Public packages:     $(@($catalog.packages).Count)" -ForegroundColor Green
        Write-Host "Public resources:    $(@($catalog.resources).Count)" -ForegroundColor Green
        Write-Host "Availability result: $($availability.available)" -ForegroundColor Green
        Write-Host "`nRentStage v0.9 public catalog read-only smoke test passed." -ForegroundColor Green
        exit 0
    }

    Write-Step "Submitting an anonymous quote request"
    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $submissionHeaders = @{
        "X-CSRF-Token" = $csrfResponse.csrf_token
        # Documentation-only TEST-NET address. A random final octet keeps local reruns
        # from consuming the same five-request/hour anti-abuse bucket.
        "X-Forwarded-For" = "198.51.100.$(Get-Random -Minimum 1 -Maximum 250)"
    }
    $requestBody = @{
        first_name = "Smoke"
        last_name = "Catalog"
        phone = "+50370000000"
        email = "rentstage-smoke-$stamp@example.com"
        company_name = "RentStage Validation"
        preferred_language = "es"
        event_type = "Prueba de catálogo público"
        event_location = "San Salvador"
        start_at = $start.ToUniversalTime().ToString("o")
        end_at = $end.ToUniversalTime().ToString("o")
        notes = "Solicitud creada por scripts/smoke-public-catalog.ps1."
        consent_accepted = $true
        website = ""
        selections = $selection
    } | ConvertTo-Json -Depth 8

    $receipt = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/public/catalogs/$TenantSlug/quote-requests" `
        -WebSession $webSession `
        -Headers $submissionHeaders `
        -ContentType "application/json" `
        -Body $requestBody

    if (-not ($receipt.reference_code -match '^RQ-[A-F0-9]{12}$')) {
        throw "The quote request did not return a valid public reference code."
    }

    Write-Step "Signing in as the smoke-test owner"
    $signIn = Invoke-RentStageFirebasePasswordSignIn `
        -Mode $AuthMode `
        -AuthBase $AuthBase `
        -ApiKey $ApiKey `
        -Email $Email `
        -Password $Password

    if (-not $signIn.idToken) {
        throw "Firebase Authentication did not return an ID token."
    }

    $sessionBody = @{ id_token = $signIn.idToken } | ConvertTo-Json
    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $sessionBody | Out-Null

    Write-Step "Confirming the request in the authenticated inbox"
    $me = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $webSession

    if (@($me.permissions) -notcontains "quote_request.read") {
        throw "The authenticated role does not expose quote_request.read."
    }

    $encodedReference = [Uri]::EscapeDataString($receipt.reference_code)
    $requestList = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/quote-requests?q=$encodedReference" `
        -WebSession $webSession

    $request = @($requestList.items) | Where-Object { $_.reference_code -eq $receipt.reference_code } | Select-Object -First 1
    if (-not $request) {
        throw "The anonymous quote request was not found in the tenant inbox."
    }

    $detail = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/quote-requests/$($request.id)" `
        -WebSession $webSession

    if (@($detail.packages).Count -ne 1 -or @($detail.items).Count -lt 1) {
        throw "The request did not preserve its package and quote-line snapshots."
    }
    if (-not $detail.terms_text -or -not $detail.terms_version -or -not $detail.consent_accepted) {
        throw "The request did not preserve the accepted terms snapshot."
    }

    Write-Step "Marking the smoke request as spam to keep the active inbox clean"
    $statusBody = @{ status = "SPAM" } | ConvertTo-Json
    $updated = Invoke-RestMethod `
        -Method Patch `
        -Uri "$ApiBase/api/v1/quote-requests/$($request.id)" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $statusBody

    if ($updated.status -ne "SPAM") {
        throw "The smoke request status did not change to SPAM."
    }

    Write-Step "Closing the server session"
    Invoke-RestMethod `
        -Method Delete `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers | Out-Null

    Write-Host "Tenant:              $($catalog.tenant.name)" -ForegroundColor Green
    Write-Host "Public package:      $($package.name)" -ForegroundColor Green
    Write-Host "Availability result: $($availability.available)" -ForegroundColor Green
    Write-Host "Request reference:   $($receipt.reference_code)" -ForegroundColor Green
    Write-Host "Inbox status:        $($updated.status)" -ForegroundColor Green
    Write-Host "`nRentStage v0.9 public catalog smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage Public Catalog smoke test failed: $($_.Exception.Message)"
    exit 1
}
