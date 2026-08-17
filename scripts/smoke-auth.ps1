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

    Write-Step "Exchanging the ID token for an HttpOnly RentStage session"
    $sessionBody = @{ id_token = $signIn.idToken } | ConvertTo-Json
    $me = Invoke-RestMethod `
        -Method Post `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers `
        -ContentType "application/json" `
        -Body $sessionBody

    Write-Step "Reading the authenticated user and active workspace"
    $authenticatedMe = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/auth/me" `
        -WebSession $webSession

    Write-Step "Reading a tenant-protected endpoint"
    $dashboard = Invoke-RestMethod `
        -Method Get `
        -Uri "$ApiBase/api/v1/dashboard" `
        -WebSession $webSession

    Write-Host "Authenticated user: $($authenticatedMe.user.email)" -ForegroundColor Green
    Write-Host "Active workspace:   $($authenticatedMe.active_workspace.name)" -ForegroundColor Green
    Write-Host "Role:               $($authenticatedMe.active_workspace.role)" -ForegroundColor Green
    Write-Host "Permissions:        $($authenticatedMe.permissions.Count)" -ForegroundColor Green
    Write-Host "Dashboard response: OK" -ForegroundColor Green

    Write-Step "Closing the server session"
    Invoke-RestMethod `
        -Method Delete `
        -Uri "$ApiBase/api/v1/auth/session" `
        -WebSession $webSession `
        -Headers $headers | Out-Null

    Write-Host "`nRentStage authentication smoke test passed." -ForegroundColor Green
}
catch {
    Write-Error "RentStage authentication smoke test failed: $($_.Exception.Message)"
    exit 1
}
