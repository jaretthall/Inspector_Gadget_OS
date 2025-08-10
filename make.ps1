# Inspector Gadget OS - Phase 0 Build System (PowerShell)
param(
    [string]$Target = "all"
)

function Show-Help {
    Write-Host "Inspector Gadget OS - Phase 0 Build Commands" -ForegroundColor Cyan
    Write-Host ""
    Write-Host "Usage: .\make.ps1 [target]"
    Write-Host ""
    Write-Host "Available targets:" -ForegroundColor Yellow
    Write-Host "  all           - Build all modules (default)"
    Write-Host "  gadget-cli    - Build gadget-framework CLI"
    Write-Host "  web-build     - Build web management server"
    Write-Host "  o-llama-build - Build o-llama enhanced server"
    Write-Host "  test          - Run tests for all modules"
    Write-Host "  clean         - Remove build artifacts"
    Write-Host "  help          - Show this help message"
}

function Ensure-BinDirectory {
    if (-not (Test-Path "bin")) {
        New-Item -ItemType Directory -Path "bin" | Out-Null
    }
}

function Build-GadgetCLI {
    Write-Host "Building gadget-framework..." -ForegroundColor Green
    Ensure-BinDirectory
    Set-Location "gadget-framework"
    go build -o "../bin/go-go-gadget.exe" ./cmd/go-go-gadget
    Set-Location ".."
    if ($LASTEXITCODE -eq 0) {
        Write-Host " gadget-framework built successfully" -ForegroundColor Green
    } else {
        Write-Host " gadget-framework build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-WebServer {
    Write-Host "Building web management server..." -ForegroundColor Green
    Ensure-BinDirectory
    Set-Location "web"
    go build -o "../bin/web-server.exe" .
    Set-Location ".."
    if ($LASTEXITCODE -eq 0) {
        Write-Host " web server built successfully" -ForegroundColor Green
    } else {
        Write-Host " web server build failed" -ForegroundColor Red
        exit 1
    }
}

function Build-OLlamaServer {
    Write-Host "Building o-llama enhanced server..." -ForegroundColor Green
    Ensure-BinDirectory
    Set-Location "o-llama"
    go build -o "../bin/ollama-server.exe" ./cmd/ollama-server
    Set-Location ".."
    if ($LASTEXITCODE -eq 0) {
        Write-Host " o-llama server built successfully" -ForegroundColor Green
    } else {
        Write-Host " o-llama server build failed" -ForegroundColor Red
        exit 1
    }
}

function Run-Tests {
    Write-Host "Running tests for all modules..." -ForegroundColor Green
    
    Write-Host "Testing gadget-framework..." -ForegroundColor Yellow
    Set-Location "gadget-framework"
    go test ./...
    $gadgetTestResult = $LASTEXITCODE
    Set-Location ".."
    
    Write-Host "Testing web module..." -ForegroundColor Yellow
    Set-Location "web"
    go test ./...
    $webTestResult = $LASTEXITCODE
    Set-Location ".."
    
    Write-Host "Testing o-llama module..." -ForegroundColor Yellow
    Set-Location "o-llama"
    go test ./...
    $llamaTestResult = $LASTEXITCODE
    Set-Location ".."
    
    if ($gadgetTestResult -eq 0 -and $webTestResult -eq 0 -and $llamaTestResult -eq 0) {
        Write-Host " All tests passed" -ForegroundColor Green
    } else {
        Write-Host " Some tests failed" -ForegroundColor Red
        exit 1
    }
}

function Clean-Build {
    Write-Host "Cleaning build artifacts..." -ForegroundColor Green
    
    if (Test-Path "bin") {
        Remove-Item -Recurse -Force "bin"
        Write-Host " Removed bin directory" -ForegroundColor Green
    }
    
    Set-Location "gadget-framework"
    go clean
    Set-Location ".."
    
    Set-Location "web"
    go clean
    Set-Location ".."
    
    Set-Location "o-llama"
    go clean
    Set-Location ".."
    
    Write-Host " Cleaned all modules" -ForegroundColor Green
}

# Main execution
switch ($Target.ToLower()) {
    "all" {
        Write-Host "Building all modules..." -ForegroundColor Cyan
        Build-GadgetCLI
        Build-WebServer
        Build-OLlamaServer
        Write-Host " All modules built successfully!" -ForegroundColor Green
    }
    "gadget-cli" { Build-GadgetCLI }
    "web-build" { Build-WebServer }
    "o-llama-build" { Build-OLlamaServer }
    "test" { Run-Tests }
    "clean" { Clean-Build }
    "help" { Show-Help }
    default {
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Write-Host ""
        Show-Help
        exit 1
    }
}