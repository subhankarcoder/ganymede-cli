<#
.SYNOPSIS
    Assists in validating and submitting Ganymede CLI manifests to microsoft/winget-pkgs.
.PARAMETER Version
    The package version to submit (default: 0.1.0).
#>
param (
    [string]$Version = '0.1.0'
)

$ErrorActionPreference = 'Stop'

Write-Host ('✦ Validating WinGet Manifests for subhankarcoder.ganymede v' + $Version + '...') -ForegroundColor Cyan

$manifestDir = "manifests/s/subhankarcoder/ganymede/$Version"
if (-not (Test-Path $manifestDir)) {
    Write-Error "Manifest directory not found: $manifestDir"
}

# 1. Run local winget validate
winget validate --manifest $manifestDir
if ($LASTEXITCODE -ne 0) {
    Write-Error 'Manifest validation failed. Please check errors above.'
}
Write-Host '✓ Manifest schema validation passed successfully!' -ForegroundColor Green

# 2. Submission options
Write-Host ''
Write-Host '✦ WinGet Submission Instructions:' -ForegroundColor Cyan
Write-Host '================================================================================' -ForegroundColor Gray
Write-Host 'You have 3 easy ways to submit this package to Microsoft:' -ForegroundColor Yellow
Write-Host ''
Write-Host 'Option 1 [Automated GitHub Actions - Recommended]:' -ForegroundColor White
Write-Host '  1. Create a GitHub Personal Access Token [Classic] with public_repo scope at:'
Write-Host '     https://github.com/settings/tokens'
Write-Host '  2. Go to your repository on GitHub: Settings -> Secrets and variables -> Actions'
Write-Host '  3. Add a New Repository Secret named WINGET_TOKEN with your PAT.'
Write-Host '  4. Push a git tag: git tag v0.1.0 and git push origin v0.1.0'
Write-Host '  5. The release workflow will automatically build, publish, and submit to WinGet!'
Write-Host ''
Write-Host 'Option 2 [Using official WingetCreate CLI]:' -ForegroundColor White
Write-Host '  Run the following commands in PowerShell:'
Write-Host '  Invoke-WebRequest -Uri https://aka.ms/wingetcreate/latest -OutFile wingetcreate.exe'
Write-Host ('  .\wingetcreate.exe submit manifests\s\subhankarcoder\ganymede\' + $Version + ' --token YOUR_GITHUB_PAT')
Write-Host ''
Write-Host 'Option 3 [Manual Pull Request via GitHub web/fork]:' -ForegroundColor White
Write-Host '  1. Fork https://github.com/microsoft/winget-pkgs'
Write-Host ('  2. Copy the folder: manifests/s/subhankarcoder/ganymede/' + $Version)
Write-Host '     into the exact same path in your fork.'
Write-Host ('  3. Create a Pull Request with title: New package: subhankarcoder.ganymede version ' + $Version)
Write-Host '================================================================================' -ForegroundColor Gray
