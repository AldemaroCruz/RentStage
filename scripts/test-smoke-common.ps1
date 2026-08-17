$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/smoke-common.ps1"

$emulator = Get-RentStageFirebaseSignInUri `
    -Mode emulator `
    -AuthBase "http://127.0.0.1:9099/" `
    -ApiKey "local-key"
if ($emulator -ne "http://127.0.0.1:9099/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=local-key") {
    throw "Unexpected emulator sign-in URI: $emulator"
}

$firebase = Get-RentStageFirebaseSignInUri `
    -Mode firebase `
    -AuthBase "https://ignored.example" `
    -ApiKey "staging-key"
if ($firebase -ne "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=staging-key") {
    throw "Unexpected Firebase sign-in URI: $firebase"
}

Write-Host "Smoke helper unit tests passed." -ForegroundColor Green
