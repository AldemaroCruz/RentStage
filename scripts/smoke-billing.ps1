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
$webSession = $null
$headers = $null
$sessionCreated = $false

function Write-Step([string]$Message) {
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Assert-Near([decimal]$Actual, [decimal]$Expected, [string]$Label) {
    if ([math]::Abs([double]($Actual - $Expected)) -gt 0.005) {
        throw "$Label expected $Expected but received $Actual."
    }
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

    $webSession = New-Object Microsoft.PowerShell.Commands.WebRequestSession
    $csrf = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/csrf" `
        -WebSession $webSession

    if (-not $csrf.csrf_token) {
        throw "The API did not return a CSRF token."
    }
    $headers = @{ "X-CSRF-Token" = $csrf.csrf_token }

    $sessionBody = @{ id_token = $signIn.idToken } | ConvertTo-Json
    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $sessionBody | Out-Null
    $sessionCreated = $true

    $me = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $webSession

    foreach ($permission in @("billing.read", "billing.manage", "payment.read", "payment.manage")) {
        if (@($me.permissions) -notcontains $permission) {
            throw "The authenticated role does not expose $permission."
        }
    }

    $currency = [string]$me.active_workspace.currency
    if ([string]::IsNullOrWhiteSpace($currency)) {
        $currency = "USD"
    }

    Write-Step "Checking billing settings and tax rules"
    $settings = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/billing/settings" `
        -WebSession $webSession

    if (-not $settings.enabled) {
        throw "Billing is disabled. Enable it in /settings/billing before running this smoke test."
    }

    $taxRulesResponse = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/billing/tax-rules" `
        -WebSession $webSession
    $taxRule = @($taxRulesResponse.items) |
        Where-Object { $_.active -and $_.is_default } |
        Select-Object -First 1
    if (-not $taxRule) {
        $taxRule = @($taxRulesResponse.items) |
            Where-Object { $_.active -and $_.category -eq "TAXABLE" } |
            Select-Object -First 1
    }
    if (-not $taxRule) {
        throw "No active taxable rule exists."
    }

    $customers = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/customers" `
        -WebSession $webSession
    $customer = @($customers.items) | Select-Object -First 1
    if (-not $customer) {
        throw "No customer exists. Create one before running this smoke test."
    }

    Write-Step "Creating and issuing an internal invoice"
    $today = (Get-Date).ToString("yyyy-MM-dd")
    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $invoiceBody = @{
        source_type = "MANUAL"
        source_id = $null
        customer_id = $customer.id
        issue_date = $today
        due_date = $today
        currency = $currency
        notes = "Temporary v0.11 billing smoke invoice ($stamp)."
        terms = "Internal validation document. This is not a DTE."
        items = @(@{
            resource_id = $null
            tax_rule_id = $taxRule.id
            description = "Billing Core smoke service"
            quantity = 1
            unit_price = 100
            discount_amount = 0
        })
    } | ConvertTo-Json -Depth 8

    $draft = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/invoices" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $invoiceBody

    if ($draft.status -ne "DRAFT" -or -not $draft.id -or @($draft.items).Count -ne 1) {
        throw "The API did not create the expected invoice draft."
    }

    $issued = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/invoices/$($draft.id)/issue" `
        -WebSession $webSession `
        -Headers $headers

    if ($issued.status -ne "ISSUED" -or -not $issued.invoice_number) {
        throw "The invoice was not issued correctly."
    }
    if ($issued.fiscal_status -notin @("NOT_READY", "READY_FOR_DTE")) {
        throw "The issued invoice returned an unexpected fiscal readiness state."
    }

    Write-Step "Recording a partial payment"
    $partialAmount = [decimal]50.00
    if ([decimal]$issued.total_amount -le $partialAmount) {
        $partialAmount = [math]::Round([decimal]$issued.total_amount / 2, 2)
    }
    if ($partialAmount -le 0) {
        throw "The issued invoice has no payable total."
    }

    $paymentBody = @{
        customer_id = $issued.customer_id
        amount = $partialAmount
        currency = $currency
        method = "BANK_TRANSFER"
        reference = "SMOKE-$stamp"
        notes = "Temporary partial payment created by smoke-billing.ps1."
        received_at = (Get-Date).ToUniversalTime().ToString("o")
        allocations = @(@{
            invoice_id = $issued.id
            amount = $partialAmount
        })
    } | ConvertTo-Json -Depth 8

    $payment = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/payments" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $paymentBody

    if ($payment.status -ne "CONFIRMED" -or @($payment.allocations).Count -ne 1) {
        throw "The partial payment was not recorded correctly."
    }

    $partiallyPaid = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/invoices/$($issued.id)" `
        -WebSession $webSession
    if ($partiallyPaid.status -ne "PARTIALLY_PAID") {
        throw "The invoice did not transition to PARTIALLY_PAID."
    }
    Assert-Near ([decimal]$partiallyPaid.paid_amount) $partialAmount "Paid amount"

    Write-Step "Voiding the payment and restoring the receivable"
    $voidPaymentBody = @{ reason = "Automatic smoke-test cleanup." } | ConvertTo-Json
    $voidedPayment = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/payments/$($payment.id)/void" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $voidPaymentBody
    if ($voidedPayment.status -ne "VOIDED") {
        throw "The payment was not voided."
    }

    $restored = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/invoices/$($issued.id)" `
        -WebSession $webSession
    if ($restored.status -ne "ISSUED") {
        throw "Voiding the payment did not restore the invoice to ISSUED."
    }
    Assert-Near ([decimal]$restored.paid_amount) ([decimal]0) "Restored paid amount"
    Assert-Near ([decimal]$restored.balance_due) ([decimal]$restored.total_amount) "Restored balance"

    Write-Step "Voiding the temporary invoice"
    $voidInvoiceBody = @{ reason = "Automatic smoke-test cleanup." } | ConvertTo-Json
    $voidedInvoice = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/invoices/$($issued.id)/void" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $voidInvoiceBody
    if ($voidedInvoice.status -ne "VOID" -or $voidedInvoice.fiscal_status -ne "VOIDED") {
        throw "The temporary invoice was not voided correctly."
    }

    Write-Step "Checking security-deposit lifecycle when a reservation exists"
    $reservations = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/reservations?limit=100" `
        -WebSession $webSession
    $reservation = @($reservations.items) |
        Where-Object { $_.status -ne "CANCELLED" } |
        Select-Object -First 1

    $depositResult = "SKIPPED (no eligible reservation)"
    if ($reservation) {
        $depositAmount = [decimal]25.00
        $depositBody = @{
            reservation_id = $reservation.id
            amount = $depositAmount
            currency = $currency
            method = "CASH"
            reference = "DEP-SMOKE-$stamp"
            notes = "Temporary deposit created by smoke-billing.ps1."
            mark_received = $true
            received_at = (Get-Date).ToUniversalTime().ToString("o")
        } | ConvertTo-Json -Depth 6

        $deposit = Invoke-RestMethod `
            -Method Post `
            -Uri "$ApiBase/api/v1/security-deposits" `
            -WebSession $webSession `
            -Headers $headers `
            -ContentType "application/json" `
            -Body $depositBody
        if ($deposit.status -ne "RECEIVED") {
            throw "The deposit was not recorded as received."
        }

        $settleBody = @{
            returned_amount = $depositAmount
            retained_amount = 0
            settled_at = (Get-Date).ToUniversalTime().ToString("o")
            reason = "Automatic smoke-test settlement."
        } | ConvertTo-Json

        $settled = Invoke-RestMethod `
            -Method Post `
            -Uri "$ApiBase/api/v1/security-deposits/$($deposit.id)/settle" `
            -WebSession $webSession `
            -Headers $headers `
            -ContentType "application/json" `
            -Body $settleBody
        if ($settled.status -ne "RETURNED") {
            throw "The security deposit did not complete the RETURNED lifecycle."
        }
        Assert-Near ([decimal]$settled.balance_amount) ([decimal]0) "Deposit balance"
        $depositResult = "$($settled.display_number) RETURNED"
    }

    Write-Step "Reading the financial dashboard"
    $dashboard = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/billing/dashboard" `
        -WebSession $webSession
    if (-not $dashboard.generated_at -or -not $dashboard.metrics) {
        throw "The financial dashboard returned an incomplete payload."
    }

    Write-Host "Invoice:           $($voidedInvoice.display_number) VOID" -ForegroundColor Green
    Write-Host "Payment:           $($voidedPayment.display_number) VOIDED" -ForegroundColor Green
    Write-Host "Tax calculation:   $($issued.tax_amount) $currency" -ForegroundColor Green
    Write-Host "Fiscal readiness:  $($issued.fiscal_status)" -ForegroundColor Green
    Write-Host "Security deposit:  $depositResult" -ForegroundColor Green
    Write-Host "Dashboard:         OK" -ForegroundColor Green
    Write-Host "`nRentStage v0.11 Billing & Payments smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage Billing & Payments smoke test failed: $($_.Exception.Message)"
    exit 1
}
finally {
    if ($sessionCreated -and $null -ne $webSession -and $null -ne $headers) {
        try {
            Invoke-RestMethod `
                -Method Delete `
                -Uri "$ApiBase/api/v1/auth/session" `
                -WebSession $webSession `
                -Headers $headers | Out-Null
        }
        catch {
            # Best-effort cleanup after the primary test result.
        }
    }
}
