# check-windows-env.ps1 - Pre-flight checks for Windows compilation
# Run this before compilation to verify environment is ready

$ErrorActionPreference = "Stop"

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "Windows Environment Check" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$allGood = $true

# Check Go installation
Write-Host "Checking Go installation..." -ForegroundColor Yellow
if (Get-Command go -ErrorAction SilentlyContinue) {
    $goVersion = go version
    Write-Host "  ✓ Go found: $goVersion" -ForegroundColor Green
} else {
    Write-Host "  ✗ Go not found in PATH" -ForegroundColor Red
    $allGood = $false
}

# Check Go version (should be 1.21+)
if (Get-Command go -ErrorAction SilentlyContinue) {
    $versionOutput = go version
    if ($versionOutput -match "go1\.(\d+)") {
        $majorVersion = [int]$matches[1]
        if ($majorVersion -ge 21) {
            Write-Host "  ✓ Go version is 1.$majorVersion (>= 1.21)" -ForegroundColor Green
        } else {
            Write-Host "  ✗ Go version 1.$majorVersion is too old (need >= 1.21)" -ForegroundColor Red
            $allGood = $false
        }
    }
}

Write-Host ""

# Check for go.mod
Write-Host "Checking project files..." -ForegroundColor Yellow
if (Test-Path "go.mod") {
    Write-Host "  ✓ go.mod found" -ForegroundColor Green
} else {
    Write-Host "  ✗ go.mod not found" -ForegroundColor Red
    $allGood = $false
}

# Check for vendor directory
if (Test-Path "vendor/modules.txt") {
    Write-Host "  ✓ vendor/modules.txt found" -ForegroundColor Green
} else {
    Write-Host "  ⚠ vendor/modules.txt not found (will run prepare-deps.ps1)" -ForegroundColor Yellow
}

Write-Host ""

# Check PowerShell execution policy
Write-Host "Checking PowerShell execution policy..." -ForegroundColor Yellow
$execPolicy = Get-ExecutionPolicy
Write-Host "  Current policy: $execPolicy" -ForegroundColor Gray
if ($execPolicy -eq "Restricted") {
    Write-Host "  ⚠ Execution policy is Restricted (batch wrapper will bypass)" -ForegroundColor Yellow
} else {
    Write-Host "  ✓ Execution policy allows script execution" -ForegroundColor Green
}

Write-Host ""

# Check for required tools
Write-Host "Checking build tools..." -ForegroundColor Yellow

# Check go-winres
if (Get-Command go-winres -ErrorAction SilentlyContinue) {
    Write-Host "  ✓ go-winres found" -ForegroundColor Green
} else {
    Write-Host "  ⚠ go-winres not found (will install)" -ForegroundColor Yellow
}

# Check Inno Setup (optional)
$innoPath = "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
if (Test-Path $innoPath) {
    Write-Host "  ✓ Inno Setup found" -ForegroundColor Green
} else {
    Write-Host "  ⚠ Inno Setup not found (packaging will be skipped)" -ForegroundColor Yellow
}

Write-Host ""

# Check line endings in PowerShell scripts
Write-Host "Checking PowerShell script line endings..." -ForegroundColor Yellow
$scripts = @("compile-windows.ps1", "prepare-deps.ps1")
foreach ($script in $scripts) {
    if (Test-Path $script) {
        $content = Get-Content $script -Raw
        if ($content -match "`r`n") {
            Write-Host "  ✓ $script has CRLF line endings" -ForegroundColor Green
        } elseif ($content -match "`n") {
            Write-Host "  ⚠ $script has LF line endings (batch wrapper will fix)" -ForegroundColor Yellow
        }
    }
}

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
if ($allGood) {
    Write-Host "✓ Environment check passed!" -ForegroundColor Green
    Write-Host "Ready to compile." -ForegroundColor Green
    exit 0
} else {
    Write-Host "✗ Environment check failed!" -ForegroundColor Red
    Write-Host "Please fix the issues above before compiling." -ForegroundColor Red
    exit 1
}

