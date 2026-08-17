[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ApiKey,

    [Parameter(Mandatory = $true)]
    [string]$Email,

    [Parameter(Mandatory = $true)]
    [string]$Password
)

$ErrorActionPreference = "Stop"
if ($Password.Length -lt 12) {
    throw "The staging smoke password must contain at least 12 characters."
}

$signupUri = "https://identitytoolkit.googleapis.com/v1/accounts:signUp?key=$ApiKey"
$signinUri = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=$ApiKey"
$body = @{
    email = $Email
    password = $Password
    returnSecureToken = $true
} | ConvertTo-Json

try {
    $created = Invoke-RestMethod -Method Post -Uri $signupUri -ContentType "application/json" -Body $body
    if (-not $created.localId) {
        throw "Identity Platform did not return a user identifier."
    }
    Write-Host "Created the staging smoke user." -ForegroundColor Green
}
catch {
    $detail = [string]$_.ErrorDetails.Message
    if ($detail -notmatch "EMAIL_EXISTS") {
        throw
    }

    $signedIn = Invoke-RestMethod -Method Post -Uri $signinUri -ContentType "application/json" -Body $body
    if (-not $signedIn.idToken) {
        throw "The staging smoke user exists but the configured credentials cannot sign in."
    }
    Write-Host "The staging smoke user already exists and the credentials are valid." -ForegroundColor Green
}
