
function Get-RentStageFirebaseSignInUri {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("emulator", "firebase")]
        [string]$Mode,

        [Parameter(Mandatory = $true)]
        [string]$AuthBase,

        [Parameter(Mandatory = $true)]
        [string]$ApiKey
    )

    if ($Mode -eq "firebase") {
        return "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=$ApiKey"
    }

    $base = $AuthBase.TrimEnd("/")
    return "$base/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=$ApiKey"
}

function Invoke-RentStageFirebasePasswordSignIn {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [ValidateSet("emulator", "firebase")]
        [string]$Mode,

        [Parameter(Mandatory = $true)]
        [string]$AuthBase,

        [Parameter(Mandatory = $true)]
        [string]$ApiKey,

        [Parameter(Mandatory = $true)]
        [string]$Email,

        [Parameter(Mandatory = $true)]
        [string]$Password
    )

    $uri = Get-RentStageFirebaseSignInUri -Mode $Mode -AuthBase $AuthBase -ApiKey $ApiKey
    $body = @{
        email = $Email
        password = $Password
        returnSecureToken = $true
    } | ConvertTo-Json

    return Invoke-RestMethod `
        -Method Post `
        -Uri $uri `
        -ContentType "application/json" `
        -Body $body
}
