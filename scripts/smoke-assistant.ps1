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
