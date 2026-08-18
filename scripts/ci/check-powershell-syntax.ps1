[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$repositoryRoot = (
    Resolve-Path (Join-Path $PSScriptRoot '..\..')
).Path

$searchDirectories = @(
    (Join-Path $repositoryRoot 'scripts'),
    (Join-Path $repositoryRoot 'scripts\ci'),
    (Join-Path $repositoryRoot 'scripts\lib')
)

$files = @(
    foreach ($directory in $searchDirectories) {
        if (Test-Path -LiteralPath $directory -PathType Container) {
            Get-ChildItem `
                -LiteralPath $directory `
                -Filter '*.ps1' `
                -File
        }
    }
)

$files = @(
    $files |
        Sort-Object -Property FullName -Unique
)

if ($files.Count -eq 0) {
    throw 'No PowerShell scripts were found for validation.'
}

$hasParseErrors = $false

foreach ($file in $files) {
    $tokens = $null
    $parseErrors = $null

    [void][System.Management.Automation.Language.Parser]::ParseFile(
        $file.FullName,
        [ref]$tokens,
        [ref]$parseErrors
    )

    $relativePath = (
        [System.IO.Path]::GetRelativePath(
            $repositoryRoot,
            $file.FullName
        )
    ).Replace('\', '/')

    foreach ($parseError in @($parseErrors)) {
        $hasParseErrors = $true

        Write-Host (
            '::error file={0},line={1},col={2}::{3}' -f
                $relativePath,
                $parseError.Extent.StartLineNumber,
                $parseError.Extent.StartColumnNumber,
                $parseError.Message
        )
    }
}

if ($hasParseErrors) {
    throw 'PowerShell syntax validation failed.'
}

Write-Host (
    'PowerShell syntax validation passed ({0} files).' -f
        $files.Count
)
