$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$root = Resolve-Path (Join-Path $PSScriptRoot "../..")
$apiDirectory = (Resolve-Path (Join-Path $root "apps/api")).Path
$image = "golang:1.26.6-alpine"

if ($null -eq (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Docker is required to regenerate Go dependency metadata with Go 1.26.6."
}

$mount = "type=bind,source=$apiDirectory,target=/src"

function Invoke-GoContainer {
    param(
        [Parameter(Mandatory = $true)]
        [string[]] $GoArguments,

        [Parameter(Mandatory = $true)]
        [string] $Description
    )

    Write-Host $Description -ForegroundColor Cyan

    $dockerArguments = @(
        "run",
        "--rm",
        "--mount", $mount,
        "--workdir", "/src",
        "--entrypoint", "/usr/local/go/bin/go",
        $image
    ) + $GoArguments

    & docker @dockerArguments
    if ($LASTEXITCODE -ne 0) {
        throw "$Description failed with exit code $LASTEXITCODE."
    }
}

Invoke-GoContainer `
    -Description "Checking the pinned Go toolchain in $image..." `
    -GoArguments @("version")

Invoke-GoContainer `
    -Description "Regenerating apps/api/go.mod and apps/api/go.sum..." `
    -GoArguments @("mod", "tidy")

Invoke-GoContainer `
    -Description "Verifying the synchronized Go module graph..." `
    -GoArguments @("mod", "verify")

$goSumPath = Join-Path $apiDirectory "go.sum"
$firebaseSum = Select-String `
    -Path $goSumPath `
    -Pattern '^firebase\.google\.com/go/v4 v4\.21\.0 h1:' `
    -Quiet

if (-not $firebaseSum) {
    throw "go.sum still does not contain the Firebase Admin SDK v4.21.0 checksum."
}

Write-Host ""
Write-Host "Go module metadata is complete and verified." -ForegroundColor Green
Write-Host "Commit both apps/api/go.mod and apps/api/go.sum before opening the pull request." -ForegroundColor Yellow

if ($null -ne (Get-Command git -ErrorAction SilentlyContinue)) {
    & git -C $root diff -- apps/api/go.mod apps/api/go.sum
}
