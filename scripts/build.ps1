<#
.SYNOPSIS
    Builds Ganymede CLI for local testing or release packaging.
.PARAMETER Version
    The version tag to bake into the binary (default: v0.1.0).
.PARAMETER Package
    Creates release ZIP archives for amd64 and arm64 with checksums.
#>
param (
    [string]$Version = 'v0.1.0',
    [switch]$Package
)

$ErrorActionPreference = 'Stop'

Write-Host ('✦ Building Ganymede CLI ' + $Version + '...') -ForegroundColor Cyan

$ldflags = '-H=windowsgui -s -w -X ganymede-cli/banner.Version=' + $Version

if (-not $Package) {
    go build -trimpath -ldflags $ldflags -o ganymede.exe .
    Write-Host '✓ Successfully built: ganymede.exe' -ForegroundColor Green
    Write-Host '  Test with: .\ganymede.exe version' -ForegroundColor Yellow
    exit 0
}

Write-Host '✦ Packaging release archives...' -ForegroundColor Cyan

New-Item -ItemType Directory -Force -Path 'dist/amd64', 'dist/arm64', 'artifacts' | Out-Null

# 1. Windows AMD64
Write-Host '  -> Compiling Windows AMD64...' -ForegroundColor Gray
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -trimpath -ldflags $ldflags -o dist/amd64/ganymede.exe .
Copy-Item -Path 'dist/amd64/ganymede.exe' -Destination 'ganymede.exe' -Force
Compress-Archive -Path 'ganymede.exe', 'README.md', 'LICENSE' -DestinationPath 'artifacts/ganymede-windows-amd64.zip' -Force
Remove-Item 'ganymede.exe' -Force

# 2. Windows ARM64
Write-Host '  -> Compiling Windows ARM64...' -ForegroundColor Gray
$env:GOOS = 'windows'
$env:GOARCH = 'arm64'
go build -trimpath -ldflags $ldflags -o dist/arm64/ganymede.exe .
Copy-Item -Path 'dist/arm64/ganymede.exe' -Destination 'ganymede.exe' -Force
Compress-Archive -Path 'ganymede.exe', 'README.md', 'LICENSE' -DestinationPath 'artifacts/ganymede-windows-arm64.zip' -Force
Remove-Item 'ganymede.exe' -Force

# 3. Compute Checksums
Write-Host '✦ Generating SHA256 Checksums...' -ForegroundColor Cyan
Set-Location artifacts
$hashes = Get-FileHash -Algorithm SHA256 *.zip | ForEach-Object {
    $_.Hash + '  ' + ($_.Path | Split-Path -Leaf)
}
$hashes | Out-File -FilePath checksums.txt -Encoding utf8
Get-Content checksums.txt
Set-Location ..

Write-Host '✓ Release assets ready in ./artifacts/' -ForegroundColor Green
