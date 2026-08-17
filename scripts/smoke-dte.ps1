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
$originalBilling = $null
$originalDTE = $null
$originalCustomer = $null
$customer = $null
$invoice = $null
$dte = $null
$cleanupComplete = $false
$demoCustomerID = "40000000-0000-0000-0000-000000000001"

function Write-Step([string]$Message) {
    Write-Host "`n==> $Message" -ForegroundColor Cyan
}

function Get-ApiFailureMessage($Record, [string]$Operation, [string]$Uri) {
    $status = "unknown"
    try {
        if ($null -ne $Record.Exception.Response.StatusCode) {
            $status = [int]$Record.Exception.Response.StatusCode
        }
    }
    catch {}

    $responseBody = ""
    if ($null -ne $Record.ErrorDetails -and -not [string]::IsNullOrWhiteSpace([string]$Record.ErrorDetails.Message)) {
        $responseBody = [string]$Record.ErrorDetails.Message
    }
    if ([string]::IsNullOrWhiteSpace($responseBody)) {
        try {
            $stream = $Record.Exception.Response.GetResponseStream()
            if ($null -ne $stream) {
                $reader = New-Object System.IO.StreamReader($stream)
                $responseBody = $reader.ReadToEnd()
                $reader.Dispose()
            }
        }
        catch {}
    }

    $serverDetail = $responseBody
    if (-not [string]::IsNullOrWhiteSpace($responseBody)) {
        try {
            $parsed = $responseBody | ConvertFrom-Json
            $parts = @()
            if ($parsed.error) { $parts += "error=$($parsed.error)" }
            if ($parsed.message) { $parts += "message=$($parsed.message)" }
            if ($parsed.fields) {
                $fieldParts = @($parsed.fields.PSObject.Properties | ForEach-Object {
                    "$($_.Name): $($_.Value)"
                })
                if ($fieldParts.Count -gt 0) { $parts += "fields=[$($fieldParts -join '; ')]" }
            }
            if ($parsed.request_id) { $parts += "request_id=$($parsed.request_id)" }
            if ($parts.Count -gt 0) { $serverDetail = $parts -join ' | ' }
        }
        catch {}
    }

    if ([string]::IsNullOrWhiteSpace($serverDetail)) {
        $serverDetail = [string]$Record.Exception.Message
    }
    return "$Operation failed (HTTP $status) at $Uri. $serverDetail"
}

function Invoke-ApiPatch([string]$Uri, [hashtable]$Body, [string]$Operation = "PATCH") {
    try {
        return Invoke-RestMethod `
            -Method Patch `
            -Uri $Uri `
            -WebSession $webSession `
            -Headers $headers `
            -ContentType "application/json" `
            -Body ($Body | ConvertTo-Json -Depth 12)
    }
    catch {
        throw (Get-ApiFailureMessage $_ $Operation $Uri)
    }
}

function Invoke-ApiPost([string]$Uri, $Body = $null, [string]$Operation = "POST") {
    try {
        $parameters = @{
            Method = "Post"
            Uri = $Uri
            WebSession = $webSession
            Headers = $headers
        }
        if ($null -ne $Body) {
            $parameters.ContentType = "application/json"
            $parameters.Body = ($Body | ConvertTo-Json -Depth 12)
        }
        return Invoke-RestMethod @parameters
    }
    catch {
        throw (Get-ApiFailureMessage $_ $Operation $Uri)
    }
}

function Billing-Input($Item) {
    return @{
        enabled = [bool]$Item.enabled
        legal_name = [string]$Item.legal_name
        trade_name = [string]$Item.trade_name
        tax_id = [string]$Item.tax_id
        tax_registration_number = [string]$Item.tax_registration_number
        economic_activity = [string]$Item.economic_activity
        economic_activity_code = [string]$Item.economic_activity_code
        fiscal_address = [string]$Item.fiscal_address
        department = [string]$Item.department
        municipality = [string]$Item.municipality
        district = [string]$Item.district
        department_code = [string]$Item.department_code
        municipality_code = [string]$Item.municipality_code
        district_code = [string]$Item.district_code
        email = [string]$Item.email
        phone = [string]$Item.phone
        prices_include_tax = [bool]$Item.prices_include_tax
        default_tax_rate = [decimal]$Item.default_tax_rate
        default_payment_terms_days = [int]$Item.default_payment_terms_days
        invoice_prefix = [string]$Item.invoice_prefix
    }
}

function DTE-Input($Item) {
    return @{
        enabled = [bool]$Item.enabled
        provider_mode = [string]$Item.provider_mode
        environment = [string]$Item.environment
        default_document_type = [string]$Item.default_document_type
        schema_version = [int]$Item.schema_version
        establishment_type = [string]$Item.establishment_type
        establishment_code = [string]$Item.establishment_code
        point_of_sale_code = [string]$Item.point_of_sale_code
        auth_url = [string]$Item.auth_url
        signer_url = [string]$Item.signer_url
        reception_url = [string]$Item.reception_url
        invalidation_url = [string]$Item.invalidation_url
        query_url = [string]$Item.query_url
        user_secret_ref = [string]$Item.user_secret_ref
        password_secret_ref = [string]$Item.password_secret_ref
        signing_password_secret_ref = [string]$Item.signing_password_secret_ref
        auto_submit_on_issue = $false
        max_attempts = [int]$Item.max_attempts
        retry_base_seconds = [int]$Item.retry_base_seconds
    }
}

function Customer-Input($Item) {
    return @{
        first_name = [string]$Item.first_name
        last_name = [string]$Item.last_name
        phone = $Item.phone
        email = $Item.email
        company_name = $Item.company_name
        tax_id = [string]$Item.tax_id
        tax_registration_number = [string]$Item.tax_registration_number
        billing_address = [string]$Item.billing_address
        document_type_code = [string]$Item.document_type_code
        trade_name = [string]$Item.trade_name
        economic_activity = [string]$Item.economic_activity
        economic_activity_code = [string]$Item.economic_activity_code
        department_code = [string]$Item.department_code
        municipality_code = [string]$Item.municipality_code
        district_code = [string]$Item.district_code
        preferred_language = [string]$Item.preferred_language
        source = [string]$Item.source
        notes = [string]$Item.notes
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
    $csrf = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/auth/csrf" -WebSession $webSession
    if (-not $csrf.csrf_token) {
        throw "The API did not return a CSRF token."
    }
    $headers = @{ "X-CSRF-Token" = $csrf.csrf_token }

    Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ id_token = $signIn.idToken } | ConvertTo-Json) | Out-Null
    $sessionCreated = $true

    $me = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/auth/me" -WebSession $webSession
    foreach ($permission in @("billing.read", "billing.manage", "fiscal.read", "fiscal.manage")) {
        if (@($me.permissions) -notcontains $permission) {
            throw "The authenticated role does not expose $permission."
        }
    }

    Write-Step "Saving current fiscal configuration"
    $originalBilling = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/billing/settings" -WebSession $webSession
    $originalDTE = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/dte-settings" -WebSession $webSession

    $customers = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/customers" -WebSession $webSession
    $customerSummary = @($customers.items) |
        Where-Object { $_.id -eq $demoCustomerID } |
        Select-Object -First 1
    if (-not $customerSummary) {
        $customerSummary = @($customers.items) |
            Where-Object { [string]$_.email -eq "carlos@example.com" } |
            Select-Object -First 1
    }
    if (-not $customerSummary) {
        throw "The deterministic AudioPro demo customer was not found. Expected ID $demoCustomerID or carlos@example.com."
    }
    $originalCustomer = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/customers/$($customerSummary.id)" -WebSession $webSession
    $customer = $originalCustomer

    Write-Step "Preparing a complete MOCK issuer and receiver profile"

    Write-Step "Updating the temporary issuer billing profile"
    $billingInput = Billing-Input $originalBilling
    $billingInput.enabled = $true
    $billingInput.legal_name = "AudioPro Demo, S.A. de C.V."
    $billingInput.trade_name = "AudioPro Demo"
    $billingInput.tax_id = "06140000000000"
    $billingInput.tax_registration_number = "1234567"
    $billingInput.economic_activity = "Alquiler de equipo de audio"
    $billingInput.economic_activity_code = "77290"
    $billingInput.fiscal_address = "San Salvador, El Salvador"
    $billingInput.department = "San Salvador"
    $billingInput.municipality = "San Salvador Centro"
    $billingInput.district = "San Salvador"
    $billingInput.department_code = "06"
    $billingInput.municipality_code = "23"
    $billingInput.district_code = "01"
    $billingInput.email = "facturacion@example.com"
    $billingInput.phone = "+50370000000"
    $billingInput.invoice_prefix = "INV"
    $billingInput.default_tax_rate = 13
    $billingInput.default_payment_terms_days = 0
    $billing = Invoke-ApiPatch "$ApiBase/api/v1/billing/settings" $billingInput "Issuer billing-profile update"
    if (-not $billing.fiscal_profile_complete) {
        throw "The temporary issuer fiscal profile is still incomplete: $($billing.fiscal_profile_missing_fields -join ', ')."
    }

    Write-Step "Updating the deterministic demo customer fiscal profile"
    $customerInput = Customer-Input $originalCustomer
    $customerInput.first_name = "Carlos"
    $customerInput.last_name = "Hernández"
    $customerInput.phone = "+50371234567"
    $customerInput.email = "carlos@example.com"
    $customerInput.company_name = $null
    $customerInput.tax_id = "06140000000001"
    $customerInput.tax_registration_number = "7654321"
    $customerInput.billing_address = "San Salvador, El Salvador"
    $customerInput.document_type_code = "36"
    $customerInput.trade_name = "Carlos Hernández"
    $customerInput.economic_activity = "Servicios para eventos"
    $customerInput.economic_activity_code = "90000"
    $customerInput.department_code = "06"
    $customerInput.municipality_code = "23"
    $customerInput.district_code = "01"
    $customerInput.preferred_language = "es"
    $customerInput.source = "MANUAL"
    $customer = Invoke-ApiPatch "$ApiBase/api/v1/customers/$($originalCustomer.id)" $customerInput "Demo customer fiscal-profile update"

    Write-Step "Enabling the MOCK / TEST DTE provider"
    $dteInput = @{
        enabled = $true
        provider_mode = "MOCK"
        environment = "TEST"
        default_document_type = "01"
        schema_version = 1
        establishment_type = "01"
        establishment_code = "M001"
        point_of_sale_code = "P001"
        auth_url = ""
        signer_url = ""
        reception_url = ""
        invalidation_url = ""
        query_url = ""
        user_secret_ref = ""
        password_secret_ref = ""
        signing_password_secret_ref = ""
        auto_submit_on_issue = $false
        max_attempts = 5
        retry_base_seconds = 60
    }
    $dteSettings = Invoke-ApiPatch "$ApiBase/api/v1/dte-settings" $dteInput "MOCK DTE settings update"
    if (-not $dteSettings.configuration_ready -or $dteSettings.provider_mode -ne "MOCK" -or $dteSettings.environment -ne "TEST") {
        throw "The MOCK DTE provider was not configured correctly."
    }

    Write-Step "Creating and issuing a temporary invoice"
    $taxRules = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/billing/tax-rules" -WebSession $webSession
    $taxRule = @($taxRules.items) | Where-Object { $_.active -and $_.category -eq "TAXABLE" } | Select-Object -First 1
    if (-not $taxRule) {
        throw "No active taxable rule exists."
    }
    $today = (Get-Date).ToString("yyyy-MM-dd")
    $stamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
    $invoiceBody = @{
        source_type = "MANUAL"
        source_id = $null
        customer_id = $customer.id
        issue_date = $today
        due_date = $today
        currency = "USD"
        notes = "Temporary v0.12 DTE smoke invoice ($stamp)."
        terms = "Documento de prueba del proveedor MOCK."
        items = @(@{
            resource_id = $null
            tax_rule_id = $taxRule.id
            description = "Servicio de renta para prueba DTE"
            quantity = 1
            unit_price = 100
            discount_amount = 0
        })
    } | ConvertTo-Json -Depth 10
    $draft = Invoke-RestMethod -Method Post -Uri "$ApiBase/api/v1/invoices" -WebSession $webSession -Headers $headers -ContentType "application/json" -Body $invoiceBody
    $invoice = Invoke-RestMethod -Method Post -Uri "$ApiBase/api/v1/invoices/$($draft.id)/issue" -WebSession $webSession -Headers $headers
    if ($invoice.status -ne "ISSUED" -or $invoice.fiscal_status -ne "READY_FOR_DTE") {
        throw "The temporary invoice is not READY_FOR_DTE."
    }

    Write-Step "Preparing an immutable DTE snapshot"
    $dte = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/invoices/$($invoice.id)/dte" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body (@{ document_type = "01" } | ConvertTo-Json)
    if ($dte.status -ne "READY_TO_SIGN") {
        throw "The DTE was not prepared in READY_TO_SIGN."
    }
    if (-not ([string]$dte.control_number).StartsWith("DTE-01-")) {
        throw "The DTE control number is malformed."
    }
    if ([string]$dte.payload.identificacion.tipoDte -ne "01") {
        throw "The payload does not contain document type 01."
    }
    $serializedPayload = $dte.payload | ConvertTo-Json -Depth 30 -Compress
    foreach ($forbidden in @("DTE_MH_PASSWORD", "DTE_MH_SIGNING_PASSWORD", "passwordPri", "access_token")) {
        if ($serializedPayload -match [regex]::Escape($forbidden)) {
            throw "The prepared DTE payload exposed forbidden secret material: $forbidden."
        }
    }

    Write-Step "Signing and submitting through the local MOCK provider"
    $dte = Invoke-ApiPost "$ApiBase/api/v1/dte/$($dte.id)/submit" $null "MOCK DTE submission"
    if ($dte.status -ne "ACCEPTED" -or -not ([string]$dte.receipt_seal).StartsWith("MOCK-")) {
        throw "The MOCK provider did not accept the DTE with a local receipt seal."
    }
    if ([int]$dte.attempt_count -ne 1) {
        throw "The DTE attempt count should be 1."
    }
    $acceptedInvoice = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/invoices/$($invoice.id)" -WebSession $webSession
    if ($acceptedInvoice.fiscal_status -ne "ACCEPTED") {
        throw "The invoice did not transition to fiscal status ACCEPTED."
    }

    Write-Step "Invalidating the accepted MOCK document"
    $dte = Invoke-ApiPost `
        "$ApiBase/api/v1/dte/$($dte.id)/invalidate" `
        @{ reason = "Automatic v0.12 smoke-test invalidation." } `
        "MOCK DTE invalidation"
    if ($dte.status -ne "INVALIDATED") {
        throw "The DTE did not complete the INVALIDATED lifecycle."
    }
    $invalidatedInvoice = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/invoices/$($invoice.id)" -WebSession $webSession
    if ($invalidatedInvoice.fiscal_status -ne "VOIDED") {
        throw "Invalidating the DTE did not mark the invoice fiscal status VOIDED."
    }

    Write-Step "Voiding the temporary internal invoice"
    $invoice = Invoke-ApiPost `
        "$ApiBase/api/v1/invoices/$($invoice.id)/void" `
        @{ reason = "Automatic v0.12 smoke-test cleanup." } `
        "Temporary invoice cleanup"
    if ($invoice.status -ne "VOID") {
        throw "The temporary invoice was not voided."
    }

    $documents = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/dte?status=INVALIDATED&limit=250" -WebSession $webSession
    if (-not (@($documents.items).id -contains $dte.id)) {
        throw "The invalidated DTE was not returned by the fiscal inbox."
    }

    $cleanupComplete = $true
    Write-Host "Invoice:            $($invoice.display_number) VOID" -ForegroundColor Green
    Write-Host "DTE control:        $($dte.control_number)" -ForegroundColor Green
    Write-Host "Generation code:    $($dte.generation_code)" -ForegroundColor Green
    Write-Host "Receipt seal:       $($dte.receipt_seal)" -ForegroundColor Green
    Write-Host "Final DTE status:   $($dte.status)" -ForegroundColor Green
    Write-Host "Provider:           MOCK / TEST" -ForegroundColor Green
    Write-Host "`nRentStage v0.12 DTE smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage DTE smoke test failed: $($_.Exception.Message)"
    exit 1
}
finally {
    # Complete the temporary fiscal lifecycle before restoring provider settings.
    if ($sessionCreated -and $null -ne $webSession -and $null -ne $headers) {
        if ($null -ne $dte -and -not $cleanupComplete) {
            try {
                $currentDTE = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/dte/$($dte.id)" -WebSession $webSession
                if ($currentDTE.status -in @("READY_TO_SIGN", "RETRY_REQUIRED")) {
                    Invoke-RestMethod -Method Post -Uri "$ApiBase/api/v1/dte/$($currentDTE.id)/cancel" -WebSession $webSession -Headers $headers | Out-Null
                }
                elseif ($currentDTE.status -eq "ACCEPTED") {
                    Invoke-ApiPost `
                        "$ApiBase/api/v1/dte/$($currentDTE.id)/invalidate" `
                        @{ reason = "Best-effort smoke-test cleanup." } `
                        "Best-effort DTE invalidation" | Out-Null
                }
                elseif ($currentDTE.status -eq "SUBMITTING") {
                    Write-Warning "DTE $($currentDTE.id) remains SUBMITTING. Restart the patched API so migration 012 can recover MOCK / TEST submissions."
                }
            }
            catch {
                # Preserve the primary test error; the document remains visible in /dte.
            }
        }

        if ($null -ne $invoice -and $invoice.id) {
            try {
                $currentInvoice = Invoke-RestMethod -Method Get -Uri "$ApiBase/api/v1/invoices/$($invoice.id)" -WebSession $webSession
                if ($currentInvoice.status -ne "VOID") {
                    Invoke-RestMethod `
                        -Method Post `
                        -Uri "$ApiBase/api/v1/invoices/$($invoice.id)/void" `
                        -WebSession $webSession `
                        -Headers $headers `
                        -ContentType "application/json" `
                        -Body (@{ reason = "Best-effort smoke-test cleanup." } | ConvertTo-Json) | Out-Null
                }
            }
            catch {
                # The temporary invoice remains auditable when cleanup cannot proceed.
            }
        }

        if ($null -ne $originalCustomer) {
            try { Invoke-ApiPatch "$ApiBase/api/v1/customers/$($originalCustomer.id)" (Customer-Input $originalCustomer) "Restore original customer" | Out-Null } catch {}
        }
        if ($null -ne $originalBilling) {
            try { Invoke-ApiPatch "$ApiBase/api/v1/billing/settings" (Billing-Input $originalBilling) "Restore original billing settings" | Out-Null } catch {}
        }
        if ($null -ne $originalDTE) {
            try { Invoke-ApiPatch "$ApiBase/api/v1/dte-settings" (DTE-Input $originalDTE) "Restore original DTE settings" | Out-Null } catch {}
        }

        try {
            Invoke-RestMethod -Method Delete -Uri "$ApiBase/api/v1/auth/session" -WebSession $webSession -Headers $headers | Out-Null
        }
        catch {
            # Best-effort logout.
        }
    }
}
