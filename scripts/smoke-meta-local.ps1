[CmdletBinding()]
param(
    [string]$ApiBaseUrl = 'http://127.0.0.1:8080',
    [string]$VerifyToken = 'rentstage-local-meta-verify-token',
    [string]$AppSecret = 'rentstage-local-meta-app-secret',
    [string]$AccessToken = 'rentstage-local-meta-access-token',
    [string]$PhoneNumberId = '100000000000001',
    [string]$WabaId = '200000000000001'
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$apiRoot = $ApiBaseUrl.TrimEnd('/')
$challenge = [Guid]::NewGuid().ToString('N')
$verifyUri = (
    '{0}/api/v1/integrations/meta/webhook?hub.mode=subscribe&hub.verify_token={1}&hub.challenge={2}' -f
        $apiRoot,
        [Uri]::EscapeDataString($VerifyToken),
        [Uri]::EscapeDataString($challenge)
)

$verification = Invoke-WebRequest `
    -Uri $verifyUri `
    -Method Get

if (
    $verification.StatusCode -ne 200 -or
    $verification.Content.Trim() -ne $challenge
) {
    throw 'Meta webhook verification did not return the raw challenge.'
}

$messageId = 'wamid.local.inbound.{0}' -f (
    [Guid]::NewGuid().ToString('N')
)
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()
$payload = [ordered]@{
    object = 'whatsapp_business_account'
    entry = @(
        [ordered]@{
            id = $WabaId
            changes = @(
                [ordered]@{
                    field = 'messages'
                    value = [ordered]@{
                        messaging_product = 'whatsapp'
                        metadata = [ordered]@{
                            display_phone_number = '+503 7000-0000'
                            phone_number_id = $PhoneNumberId
                        }
                        contacts = @(
                            [ordered]@{
                                wa_id = '50370123456'
                                profile = [ordered]@{
                                    name = 'Cliente Meta Local'
                                }
                            }
                        )
                        messages = @(
                            [ordered]@{
                                from = '50370123456'
                                id = $messageId
                                timestamp = $timestamp
                                type = 'text'
                                text = [ordered]@{
                                    body = 'Hola, necesito una cotización desde el webhook local de Meta.'
                                }
                            }
                        )
                    }
                }
            )
        }
    )
}
$json = $payload |
    ConvertTo-Json -Depth 12 -Compress
$bodyBytes = [Text.Encoding]::UTF8.GetBytes($json)
$secretBytes = [Text.Encoding]::UTF8.GetBytes($AppSecret)
$hmac = [Security.Cryptography.HMACSHA256]::new($secretBytes)

try {
    $digest = $hmac.ComputeHash($bodyBytes)
}
finally {
    $hmac.Dispose()
}

$signature = 'sha256={0}' -f (
    [Convert]::ToHexString($digest).ToLowerInvariant()
)
$webhookUri = '{0}/api/v1/integrations/meta/webhook' -f $apiRoot
$headers = @{
    'X-Hub-Signature-256' = $signature
}

$first = Invoke-RestMethod `
    -Uri $webhookUri `
    -Method Post `
    -Headers $headers `
    -ContentType 'application/json' `
    -Body $bodyBytes

if ($first.inbound_processed -ne 1) {
    throw 'The first signed webhook was not processed exactly once.'
}

$duplicate = Invoke-RestMethod `
    -Uri $webhookUri `
    -Method Post `
    -Headers $headers `
    -ContentType 'application/json' `
    -Body $bodyBytes

if ($duplicate.duplicates -ne 1) {
    throw 'Webhook idempotency did not identify the duplicate message.'
}

$graphUri = (
    '{0}/api/v1/integrations/meta/local-graph/v-test/{1}/messages' -f
        $apiRoot,
        $PhoneNumberId
)
$graphPayload = [ordered]@{
    messaging_product = 'whatsapp'
    recipient_type = 'individual'
    to = '50370123456'
    type = 'text'
    text = [ordered]@{
        preview_url = $false
        body = 'Respuesta contractual de RentStage en Meta local.'
    }
} |
    ConvertTo-Json -Depth 5 -Compress
$graph = Invoke-RestMethod `
    -Uri $graphUri `
    -Method Post `
    -Headers @{ Authorization = "Bearer $AccessToken" } `
    -ContentType 'application/json' `
    -Body $graphPayload

if (
    @($graph.messages).Count -ne 1 -or
    -not $graph.messages[0].id.StartsWith('wamid.local.')
) {
    throw 'The local Graph API did not return a provider message ID.'
}

Write-Host 'Meta local contract passed:' -ForegroundColor Green
Write-Host '  webhook verification: raw challenge accepted'
Write-Host '  webhook signature: HMAC SHA-256 accepted'
Write-Host '  inbound processing: exactly once'
Write-Host '  duplicate delivery: ignored safely'
Write-Host '  Graph send contract: local provider ID returned'
Write-Host ''
Write-Host 'Open http://127.0.0.1:3000/assistant and select Cliente Meta Local.'
Write-Host 'Sending from that conversation stays inside the local Graph harness.'
