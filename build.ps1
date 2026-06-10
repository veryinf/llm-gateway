# LLM Gateway Build Script
# Usage: .\build.ps1 [command]
# Commands: build, run, dev, test, clean, fmt, vet, help

param(
    [Parameter(Position = 0)]
    [string]$Command = "help",
    [switch]$Help
)

if ($Help) { $Command = "help" }

$BinaryName = "llm-gateway.exe"
$BuildDir = "output"
$DataDir = "data"
$WebDir = "web"

function Write-Step { param([string]$msg) Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-OK { param([string]$msg) Write-Host "√ $msg" -ForegroundColor Green }
function Write-Err { param([string]$msg) Write-Host "× $msg" -ForegroundColor Red }

switch ($Command) {
    "build" {
        Write-Step "Building $BinaryName..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        go build -ldflags="-s -w" -o "$BuildDir/$BinaryName" ./cmd/gateway/
        if ($LASTEXITCODE -eq 0) {
            Write-OK "Build success: $BuildDir/$BinaryName"
        } else {
            Write-Err "Build failed"
            exit $LASTEXITCODE
        }
    }

    "run" {
        Write-Step "Building and starting server..."
        New-Item -ItemType Directory -Force -Path $DataDir, $BuildDir | Out-Null
        go build -o "$BuildDir/$BinaryName" ./cmd/gateway/
        if ($LASTEXITCODE -ne 0) { Write-Err "Build failed"; exit $LASTEXITCODE }
        Write-OK "Build done, starting..."
        & "$BuildDir/$BinaryName"
    }

    "dev" {
        Write-Step "Building and starting server in dev mode..."
        New-Item -ItemType Directory -Force -Path $DataDir, $BuildDir | Out-Null
        go build -o "$BuildDir/$BinaryName" ./cmd/gateway/
        if ($LASTEXITCODE -ne 0) { Write-Err "Build failed"; exit $LASTEXITCODE }
        Write-OK "Build done, starting..."
        & "$BuildDir/$BinaryName"
    }

    "test" {
        Write-Step "Running tests..."
        go test -v -count=1 -coverprofile=coverage.out ./...
        if ($LASTEXITCODE -eq 0) { Write-OK "All tests passed" } else { Write-Err "Tests failed" }
    }

    "test-cover" {
        Write-Step "Running tests with coverage..."
        go test -v -count=1 -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
        Write-OK "Coverage report: coverage.html"
    }

    "fmt" {
        Write-Step "Formatting code..."
        go fmt ./...
        Write-OK "Format done"
    }

    "vet" {
        Write-Step "Running go vet..."
        go vet ./...
        if ($LASTEXITCODE -eq 0) { Write-OK "Vet passed" } else { Write-Err "Vet found issues" }
    }

    "deps" {
        Write-Step "Installing Go dependencies..."
        go mod download
        Write-OK "Done"
    }

    "tidy" {
        Write-Step "Tidying modules..."
        go mod tidy
        Write-OK "Done"
    }

    "clean" {
        Write-Step "Cleaning..."
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $BuildDir
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "$WebDir/dist"
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "$WebDir/node_modules"
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "cmd/gateway/dist"
        Remove-Item -Force -ErrorAction SilentlyContinue coverage.out, coverage.html
        Write-OK "Clean done"
    }

    "build-frontend" {
        Write-Step "Building frontend..."
        Push-Location $WebDir
        try {
            if (!(Test-Path "node_modules")) { pnpm install --frozen-lockfile }
            pnpm build
            if ($LASTEXITCODE -eq 0) { Write-OK "Build → dist/" } else { Write-Err "Build failed" }
        } finally { Pop-Location }
    }

    "build-all" {
        Write-Step "Building frontend..."
        Push-Location $WebDir
        try {
            if (!(Test-Path "node_modules")) { pnpm install --frozen-lockfile }
            pnpm build
        } finally { Pop-Location }
        # Copy frontend dist into cmd/gateway/dist for embedding
        Write-Step "Copying frontend dist to cmd/gateway/dist..."
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "cmd/gateway/dist"
        New-Item -ItemType Directory -Force -Path "cmd/gateway/dist" | Out-Null
        Copy-Item -Recurse "$WebDir/dist/*" "cmd/gateway/dist/"
        Write-Step "Building $BinaryName (with embedded frontend)..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        go build -ldflags="-s -w" -o "$BuildDir/$BinaryName" ./cmd/gateway/
        if ($LASTEXITCODE -eq 0) { Write-OK "Build all → $BuildDir/$BinaryName (standalone, frontend embedded)" }
    }

    "dev-frontend" {
        Write-Step "Starting frontend dev server..."
        Push-Location $WebDir
        try { pnpm dev } finally { Pop-Location }
    }

    "setup" {
        Write-Step "Installing dependencies..."
        Push-Location $WebDir
        try {
            if (Test-Path "pnpm-lock.yaml") { pnpm install --frozen-lockfile } else { pnpm install }
            Write-OK "Frontend deps installed"
        } finally { Pop-Location }
        go mod download
        Write-OK "Go deps installed"
    }

    default {
        Write-Host ""
        Write-Host "LLM Gateway Build Script" -ForegroundColor Cyan
        Write-Host "========================" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Usage: .\build.ps1 <command>"
        Write-Host ""
        Write-Host "Commands:" -ForegroundColor Yellow
        Write-Host "  setup           Install all dependencies (Go + pnpm)"
        Write-Host "  build           Build Go binary -> output/"
        Write-Host "  build-all       Build frontend + Go binary"
        Write-Host "  build-frontend  Build React 19 frontend -> web/dist/"
        Write-Host "  run             Build and run server"
        Write-Host "  dev             Build and run server"
        Write-Host "  dev-frontend    Start Vite dev server (http://localhost:5173)"
        Write-Host "  test            Run all tests"
        Write-Host "  test-cover      Run tests with coverage report"
        Write-Host "  fmt             Format Go code"
        Write-Host "  vet             Run go vet"
        Write-Host "  deps            Install Go dependencies"
        Write-Host "  tidy            Tidy Go modules"
        Write-Host "  clean           Remove build artifacts, node_modules"
        Write-Host ""
        Write-Host "Examples:" -ForegroundColor Yellow
        Write-Host "  .\build.ps1 dev          # Start backend"
        Write-Host "  .\build.ps1 dev-frontend # Start frontend (another terminal)"
        Write-Host "  .\build.ps1 build-all    # Production build"
        Write-Host "  .\build.ps1 test         # Run tests"
        Write-Host ""
    }
}
