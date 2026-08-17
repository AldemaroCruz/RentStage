[CmdletBinding()]
param(
    [string]$ApiBase = "http://127.0.0.1:8080",
    [string]$AuthBase = "http://127.0.0.1:9099",
    [ValidateSet("emulator", "firebase")]
    [string]$AuthMode = "emulator",
    [string]$ApiKey = "rentstage-local-api-key",
    [string]$Email = "owner@rentstage.local",
    [string]$Password = "RentStage123!"
)

. "$PSScriptRoot/lib/smoke-common.ps1"

$ErrorActionPreference = "Stop"

function Write-Step([string]$Message) {
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Get-HttpStatusFromError($ErrorRecord) {
    $response = $ErrorRecord.Exception.Response
    if ($null -eq $response) { return 0 }
    if ($response.StatusCode -is [int]) { return [int]$response.StatusCode }
    if ($null -ne $response.StatusCode.value__) { return [int]$response.StatusCode.value__ }
    return [int]$response.StatusCode
}

try {
    Write-Step "Checking API readiness"
    $ready = Invoke-RestMethod -Method Get -Uri "$ApiBase/readyz"
    if ($ready.status -ne "ready") {
        throw "API readiness returned an unexpected payload."
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

    $adminSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $adminCSRF = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $adminSession

    if (-not $adminCSRF.csrf_token) {
        throw "The API did not return an administrator CSRF token."
    }
    $adminHeaders = @{ "X-CSRF-Token" = $adminCSRF.csrf_token }

    $sessionBody = @{ id_token = $signIn.idToken } | ConvertTo-Json
    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $adminSession `
        -Headers $adminHeaders `
        -ContentType "application/json" `
        -Body $sessionBody | Out-Null

    $me = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $adminSession

    foreach ($permission in @("quote.manage", "reservation.manage")) {
        if (@($me.permissions) -notcontains $permission) {
            throw "The authenticated role does not expose $permission."
        }
    }

    Write-Step "Checking Quote Portal settings"
    $portalSettings = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/quote-portal-settings" `
        -WebSession $adminSession

    if (-not $portalSettings.enabled) {
        throw "Quote Portal is disabled. Enable it in /settings/quote-portal before running this smoke test."
    }
    if (-not $portalSettings.acceptance_terms_text -or -not $portalSettings.acceptance_terms_version) {
        throw "Quote Portal terms are incomplete."
    }

    Write-Step "Selecting a customer and available catalog resource"
    $customers = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/customers" `
        -WebSession $adminSession
    $customer = @($customers.items) | Select-Object -First 1
    if (-not $customer) {
        throw "No customer exists. Create one before running this smoke test."
    }

    $resources = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/resources?active=true" `
        -WebSession $adminSession
    $resource = @($resources.items) |
        Where-Object { $_.active -and [int]$_.available_asset_count -ge 1 } |
        Select-Object -First 1
    if (-not $resource) {
        throw "No active resource with an available physical unit was found."
    }

    Write-Step "Finding a free future period"
    $start = $null
    $end = $null
    for ($attempt = 0; $attempt -lt 12; $attempt++) {
        $candidateStart = (Get-Date).Date.AddDays((Get-Random -Minimum 400 -Maximum 3000)).AddHours(14)
        $candidateEnd = $candidateStart.AddHours(6)
        $availabilityBody = @{
            start_at = $candidateStart.ToUniversalTime().ToString("o")
            end_at = $candidateEnd.ToUniversalTime().ToString("o")
            items = @(@{ resource_id = $resource.id; quantity = 1 })
        } | ConvertTo-Json -Depth 6

        $availability = Invoke-RestMethod `
            -Method Post `
            -Uri "$ApiBase/api/v1/availability/check" `
            -WebSession $adminSession `
            -Headers $adminHeaders `
            -ContentType "application/json" `
            -Body $availabilityBody

        if ($availability.available) {
            $start = $candidateStart
            $end = $candidateEnd
            break
        }
    }
    if ($null -eq $start) {
        throw "Could not find a free future period for the selected resource."
    }

    Write-Step "Creating and sending a smoke-test quote"
    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $expires = (Get-Date).ToUniversalTime().AddDays(7)
    $quoteBody = @{
        customer_id = $customer.id
        start_at = $start.ToUniversalTime().ToString("o")
        end_at = $end.ToUniversalTime().ToString("o")
        event_type = "Validación Quote Portal"
        event_location = "San Salvador"
        discount_amount = 0
        extra_charges = 0
        notes = "Cotización temporal creada por scripts/smoke-quote-portal.ps1 ($stamp)."
        expires_at = $expires.ToString("o")
        items = @(@{
            resource_id = $resource.id
            description = $resource.name
            quantity = 1
            unit_price = [decimal]$resource.base_price
            discount_amount = 0
        })
    } | ConvertTo-Json -Depth 8

    $quote = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/quotes" `
        -WebSession $adminSession `
        -Headers $adminHeaders `
        -ContentType "application/json" `
        -Body $quoteBody

    $sent = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/quotes/$($quote.id)/send" `
        -WebSession $adminSession `
        -Headers $adminHeaders

    if ($sent.status -ne "SENT" -or -not $sent.portal.public_url) {
        throw "Sending the quote did not return a one-time portal URL."
    }

    [Uri]$firstURL = $sent.portal.public_url
    $firstToken = $firstURL.Fragment.TrimStart("#")
    if ($firstURL.AbsolutePath -ne "/q" -or $firstURL.Query -ne "" -or -not $firstToken) {
        throw "The portal URL must keep the bearer token only in the URL fragment."
    }

    $publicSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $publicCSRF = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $publicSession
    if (-not $publicCSRF.csrf_token) {
        throw "The API did not return an anonymous CSRF token."
    }
    $firstTokenHeaders = @{ "X-RentStage-Quote-Token" = $firstToken }

    Write-Step "Reading the anonymous quote portal"
    $publicView = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/public/quote-portal" `
        -WebSession $publicSession `
        -Headers $firstTokenHeaders

    if ($publicView.quote.quote_number -ne $sent.quote_number -or $publicView.portal.status -ne "ACTIVE") {
        throw "The public portal returned a different quote or an unexpected status."
    }
    if (@($publicView.quote.items).Count -ne 1 -or -not $publicView.portal.can_accept) {
        throw "The public quote document is incomplete or cannot be accepted."
    }

    Write-Step "Rotating the portal token"
    $rotated = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/quotes/$($quote.id)/portal/reissue" `
        -WebSession $adminSession `
        -Headers $adminHeaders

    [Uri]$rotatedURL = $rotated.portal.public_url
    $rotatedToken = $rotatedURL.Fragment.TrimStart("#")
    if (-not $rotatedToken -or $rotatedToken -eq $firstToken -or [int]$rotated.portal.revision -lt 2) {
        throw "Token rotation did not return a new revision and secret."
    }

    try {
        Invoke-WebRequest `
            -Method Get `
            -Uri "$ApiBase/api/v1/public/quote-portal" `
            -Headers $firstTokenHeaders `
            -UseBasicParsing | Out-Null
        throw "The replaced portal token unexpectedly remained valid."
    }
    catch {
        if ($_.Exception.Message -eq "The replaced portal token unexpectedly remained valid.") { throw }
        $status = Get-HttpStatusFromError $_
        if ($status -ne 404) {
            throw "The replaced portal token returned HTTP $status instead of 404."
        }
    }

    Write-Step "Accepting the quote through the anonymous portal"
    $decisionHeaders = @{
        "X-CSRF-Token" = $publicCSRF.csrf_token
        "X-RentStage-Quote-Token" = $rotatedToken
    }
    $decisionBody = @{
        response_name = "Quote Portal Smoke"
        response_email = "quote-portal-smoke@example.com"
        terms_accepted = $true
    } | ConvertTo-Json

    $decision = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/public/quote-portal/accept" `
        -WebSession $publicSession `
        -Headers $decisionHeaders `
        -ContentType "application/json" `
        -Body $decisionBody

    if ($decision.status -ne "ACCEPTED" -or -not $decision.reservation_number -or $decision.idempotent) {
        throw "The first customer acceptance did not create a reservation."
    }

    Write-Step "Confirming idempotency and the protected quote state"
    $retry = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/public/quote-portal/accept" `
        -WebSession $publicSession `
        -Headers $decisionHeaders `
        -ContentType "application/json" `
        -Body $decisionBody
    if (-not $retry.idempotent -or $retry.reservation_number -ne $decision.reservation_number) {
        throw "Repeated acceptance was not idempotent."
    }

    $adminQuote = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/quotes/$($quote.id)" `
        -WebSession $adminSession

    if ($adminQuote.status -ne "ACCEPTED" -or -not $adminQuote.reservation_id) {
        throw "The protected quote did not preserve the online acceptance and reservation link."
    }
    if ($adminQuote.portal.status -ne "ACCEPTED" -or $adminQuote.portal.decision_source -ne "CUSTOMER") {
        throw "The portal evidence does not identify the customer decision."
    }
    if ($adminQuote.portal.response_name -ne "Quote Portal Smoke") {
        throw "The response name was not preserved in the portal evidence."
    }

    Write-Step "Cancelling the temporary reservation to release capacity"
    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/reservations/$($adminQuote.reservation_id)/cancel" `
        -WebSession $adminSession `
        -Headers $adminHeaders | Out-Null

    Write-Step "Closing the administrator session"
    Invoke-RestMethod `
        -Method Delete `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $adminSession `
        -Headers $adminHeaders | Out-Null

    Write-Host "Quote:              QT-$(([string]$sent.quote_number).PadLeft(6, '0'))" -ForegroundColor Green
    Write-Host "Portal revision:    $($rotated.portal.revision)" -ForegroundColor Green
    Write-Host "Customer decision:  $($decision.status)" -ForegroundColor Green
    Write-Host "Reservation number: $($decision.reservation_number)" -ForegroundColor Green
    Write-Host "Token transport:    URL fragment -> request header" -ForegroundColor Green
    Write-Host "`nRentStage v0.10 Quote Portal smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage Quote Portal smoke test failed: $($_.Exception.Message)"
    exit 1
}
