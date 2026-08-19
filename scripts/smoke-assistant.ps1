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

try {
    Write-Step "Signing in for the assistant flow"
    $signIn = Invoke-RentStageFirebasePasswordSignIn `
        -Mode $AuthMode `
        -AuthBase $AuthBase `
        -ApiKey $ApiKey `
        -Email $Email `
        -Password $Password

    $webSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $csrfResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $webSession
    $headers = @{ "X-CSRF-Token" = $csrfResponse.csrf_token }

    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ id_token = $signIn.idToken } | ConvertTo-Json) | Out-Null

    $me = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $webSession
    if (@($me.permissions) -notcontains "assistant.manage") {
        throw "The owner smoke user does not expose assistant.manage."
    }

    Write-Step "Selecting an existing customer for human approval"
    $customerResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/customers" `
        -WebSession $webSession
    $customer = @($customerResponse.items) | Select-Object -First 1
    if (-not $customer) {
        throw "At least one customer is required for assistant approval."
    }

    Write-Step "Simulating a WhatsApp-style inquiry"
    $start = (Get-Date).Date.AddDays(120).AddHours(15)
    $end = $start.AddHours(8)
    $simulationBody = @{
        contact_name = "Cliente Smoke"
        contact_phone = "+50370112233"
        message = "Hola, necesito sonido para una boda de 100 personas en San Salvador."
        event_type = "Boda"
        event_location = "San Salvador"
        guest_count = 100
        start_at = $start.ToUniversalTime().ToString("o")
        end_at = $end.ToUniversalTime().ToString("o")
    } | ConvertTo-Json

    $conversation = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/simulate" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $simulationBody

    if ($conversation.channel -ne "DEMO") {
        throw "The no-credential assistant must identify its channel as DEMO."
    }
    if ($conversation.status -ne "HUMAN_REVIEW") {
        throw "The simulated conversation did not stop for human review."
    }
    if (-not $conversation.proposal -or $conversation.proposal.provider -ne "DEMO_RULES") {
        throw "The deterministic proposal was not created."
    }
    if (-not $conversation.proposal.available) {
        throw "The future smoke period should have package availability."
    }

    Write-Step "Linking the conversation to a tenant-scoped customer"
    $linked = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/customer" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ customer_id = $customer.id } | ConvertTo-Json)
    if ($linked.customer_id -ne $customer.id) {
        throw "The assistant conversation was not linked to the selected customer."
    }

    Write-Step "Approving the response and creating a quote draft"
    $approvalBody = @{
        customer_id = $customer.id
        response_body = $conversation.proposal.response_draft
    } | ConvertTo-Json

    $approved = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/approve" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $approvalBody

    if ($approved.status -ne "QUOTE_DRAFTED" -or -not $approved.proposal.quote_id) {
        throw "Assistant approval did not link a quote draft."
    }

    $quote = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/quotes/$($approved.proposal.quote_id)" `
        -WebSession $webSession
    if ($quote.status -ne "DRAFT") {
        throw "Assistant approval must create a DRAFT quote."
    }
    if ($quote.reservation_id) {
        throw "Assistant approval must not create a reservation."
    }

    Write-Step "Delivering the approved response inside the demo channel"
    $pendingMessages = @(
        $approved.messages |
            Where-Object {
                $_.direction -eq "OUTBOUND" -and
                $_.status -eq "APPROVED"
            }
    )
    if ($pendingMessages.Count -eq 0) {
        throw "The approved assistant response was not available for demo delivery."
    }
    $pendingMessage = $pendingMessages[-1]
    $sent = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/messages/send-demo" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{
            message_id = $pendingMessage.id
            body = $pendingMessage.body
        } | ConvertTo-Json)
    $delivered = @(
        $sent.messages |
            Where-Object {
                $_.id -eq $pendingMessage.id -and
                $_.status -eq "SENT"
            }
    )
    if ($delivered.Count -ne 1) {
        throw "The approved response was not marked as delivered in the demo channel."
    }

    Write-Step "Simulating a customer follow-up and reviewing the generated draft"
    $followUp = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/messages/receive-demo" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ body = "¿Puedo pagar un anticipo para confirmar la fecha?" } | ConvertTo-Json)
    $followUpDrafts = @(
        $followUp.messages |
            Where-Object {
                $_.direction -eq "OUTBOUND" -and
                $_.status -eq "DRAFT"
            }
    )
    if ($followUp.status -ne "HUMAN_REVIEW" -or $followUpDrafts.Count -eq 0) {
        throw "The customer follow-up did not stop at a new human-review draft."
    }

    $followUpDraft = $followUpDrafts[-1]
    $followUpSent = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/messages/send-demo" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{
            message_id = $followUpDraft.id
            body = $followUpDraft.body
        } | ConvertTo-Json)
    if ($followUpSent.status -ne "QUOTE_DRAFTED") {
        throw "The delivered follow-up did not preserve the quote-drafted conversation state."
    }

    Write-Step "Sharing the quote portal without persisting its bearer token"
    $shared = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)/quote/share-demo" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ body = "Revisa la cotización en el portal seguro y registra allí tu decisión." } | ConvertTo-Json)

    if ($shared.proposal.quote_status -ne "SENT" -or $shared.proposal.portal_status -ne "ACTIVE") {
        throw "Sharing from the assistant did not activate the quote portal."
    }
    if (-not $shared.portal_delivery.public_url) {
        throw "The assistant did not return the one-time customer portal URL."
    }

    [Uri]$portalURL = $shared.portal_delivery.public_url
    $portalToken = $portalURL.Fragment.TrimStart("#")
    if ($portalURL.AbsolutePath -ne "/q" -or $portalURL.Query -ne "" -or -not $portalToken) {
        throw "The assistant portal URL must keep its bearer token only in the URL fragment."
    }

    $serializedMessages = $shared.messages | ConvertTo-Json -Depth 12 -Compress
    if ($serializedMessages.Contains($portalToken) -or $serializedMessages.Contains($shared.portal_delivery.public_url)) {
        throw "The raw quote portal credential leaked into the assistant transcript."
    }
    $portalEvidence = @(
        $shared.messages |
            Where-Object { $_.metadata.message_kind -eq "QUOTE_PORTAL" }
    )
    if ($portalEvidence.Count -eq 0 -or $portalEvidence[-1].metadata.raw_token_persisted -ne $false) {
        throw "The assistant did not record sanitized portal-delivery evidence."
    }

    $publicHeaders = @{ "X-RentStage-Quote-Token" = $portalToken }
    $publicView = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/public/quote-portal" `
        -WebSession $webSession `
        -Headers $publicHeaders
    if ($publicView.portal.status -ne "ACTIVE" -or -not $publicView.portal.can_reject) {
        throw "The customer portal is not available for an explicit demo decision."
    }

    Write-Step "Rejecting explicitly as the simulated customer"
    $decisionHeaders = @{
        "X-CSRF-Token" = $csrfResponse.csrf_token
        "X-RentStage-Quote-Token" = $portalToken
    }
    $decision = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/public/quote-portal/reject" `
        -WebSession $webSession `
        -Headers $decisionHeaders `
        -ContentType "application/json" `
        -Body (@{
            response_name = "Cliente Smoke"
            response_email = ""
            rejection_reason = "Decisión simulada para validar el flujo sin reservar inventario."
        } | ConvertTo-Json)
    if ($decision.status -ne "REJECTED" -or $decision.reservation_number) {
        throw "The explicit rejection produced an unexpected reservation or portal state."
    }

    $refreshed = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/assistant/conversations/$($conversation.id)" `
        -WebSession $webSession
    if ($refreshed.proposal.portal_status -ne "REJECTED" -or $refreshed.proposal.reservation_id) {
        throw "The assistant did not reflect the customer rejection without a reservation."
    }

    Write-Step "Closing the server session"
    Invoke-RestMethod `
        -Method Delete `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers | Out-Null

    Write-Host "`nRentStage assistant smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage assistant smoke test failed: $($_.Exception.Message)"
    exit 1
}
