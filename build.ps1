# LLM Gateway Build Script
# Usage: .\build.ps1 <command> [--logfile [path]]
# Commands: setup, build, build-all, dev, dev-frontend, fmt, vet, help

$BinaryName = "lgw.exe"
$BuildDir = "output"
$DataDir = "data"
$WebDir = "web"

# Parse $args manually (PowerShell doesn't support -- prefix in param)
$Command = "help"
$logfile = $null
$help = $false

$i = 0
while ($i -lt $args.Count) {
    switch ($args[$i]) {
        '--logfile' {
            $i++
            if ($i -lt $args.Count -and -not $args[$i].StartsWith('--')) {
                $logfile = $args[$i]
            } else {
                $logfile = "output/build.log"
                $i--
            }
        }
        '--help' { $help = $true }
        default {
            if ($Command -eq "help") { $Command = $args[$i] }
        }
    }
    $i++
}
if ($help) { $Command = "help" }

# Initialize log file if specified
if ($logfile) {
    $logDir = Split-Path -Parent $logfile
    if ($logDir -and !(Test-Path $logDir)) { New-Item -ItemType Directory -Force -Path $logDir | Out-Null }
    Set-Content -Path $logfile -Value "$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') Build started`n"
}

function Write-Tee {
    param([string]$msg, [string]$color)
    if ($color) { Write-Host $msg -ForegroundColor $color } else { Write-Host $msg }
    if ($logfile) { Add-Content -Path $logfile -Value $msg }
}

function Run {
    param([string]$cmd)
    if ($logfile) {
        $output = cmd.exe /c "$cmd 2>&1"
        foreach ($line in $output) { Write-Tee $line }
    } else {
        Invoke-Expression $cmd
    }
}

function Write-Step { param([string]$msg) Write-Tee "==> $msg" "Cyan" }
function Write-OK { param([string]$msg) Write-Tee "√ $msg" "Green" }
function Write-Err { param([string]$msg) Write-Tee "× $msg" "Red" }

$BuildVersion = if (Test-Path env:CI_COMMIT_TAG) { $env:CI_COMMIT_TAG } else { "dev" }
$BuildTime = Get-Date -Format "2006-01-02 15:04:05"
$BuildEnv = if ($Command -eq "build" -or $Command -eq "build-all") { "production" } else { "development" }
$LdFlags = "-s -w -X 'llm-gateway/internal/core.BuildEnv=$BuildEnv' -X 'llm-gateway/internal/core.BuildTime=$BuildTime' -X 'llm-gateway/internal/core.BuildVersion=$BuildVersion'"

switch ($Command) {
    "setup" {
        Write-Step "Installing Go dependencies..."
        Run "go mod tidy"
        Write-Step "Installing frontend dependencies..."
        Push-Location $WebDir
        try { Run "pnpm install --frozen-lockfile" } finally { Pop-Location }
        Write-OK "Setup complete"
    }

    "fmt" {
        Write-Step "Formatting Go code..."
        Run "gofmt -w ."
        Write-OK "Format complete"
    }

    "vet" {
        Write-Step "Running go vet..."
        Run "go vet ./..."
        if ($LASTEXITCODE -eq 0) { Write-OK "Vet passed" } else { Write-Err "Vet failed"; exit $LASTEXITCODE }
    }

    "build" {
        Write-Step "Building $BinaryName..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        Run "go build -ldflags=""$LdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/gateway/"
        if ($LASTEXITCODE -eq 0) {
            Write-OK "Build success: $BuildDir/$BinaryName"
        } else {
            Write-Err "Build failed"
            exit $LASTEXITCODE
        }
    }

    "dev" {
        $BinaryName = "lgw-dev.exe"
        Write-Step "Building and starting server in dev mode..."
        New-Item -ItemType Directory -Force -Path $DataDir, $BuildDir | Out-Null
        Run "go build -ldflags=""$LdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/gateway/"
        if ($LASTEXITCODE -ne 0) { Write-Err "Build failed"; exit $LASTEXITCODE }
        Write-OK "Build done, starting..."
        & "$BuildDir/$BinaryName"
    }

    "build-all" {
        Write-Step "Building frontend..."
        Push-Location $WebDir
        try {
            if (!(Test-Path "node_modules")) { Run "pnpm install --frozen-lockfile" }
            Run "pnpm build"
        } finally { Pop-Location }
        Write-Step "Copying frontend dist to static/..."
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue "static"
        New-Item -ItemType Directory -Force -Path "static" | Out-Null
        Copy-Item -Recurse "$WebDir/dist/*" "static/"
        Write-Step "Building $BinaryName (with embedded frontend)..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        Run "go build -ldflags=""$LdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/gateway/"
        if ($LASTEXITCODE -eq 0) { Write-OK "Build all → $BuildDir/$BinaryName (standalone, frontend embedded)" }
    }

    "dev-frontend" {
        Write-Step "Starting frontend dev server..."
        Push-Location $WebDir
        try { pnpm dev } finally { Pop-Location }
    }

    default {
        Write-Host ""
        Write-Host "LLM Gateway Build Script" -ForegroundColor Cyan
        Write-Host "========================" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Usage: .\build.ps1 <command> [--logfile [path]]"
        Write-Host ""
        Write-Host "Commands:" -ForegroundColor Yellow
        Write-Host "  setup          Install Go + frontend dependencies"
        Write-Host "  fmt            Format Go code"
        Write-Host "  vet            Run go vet"
        Write-Host "  build          Build Go binary -> output/"
        Write-Host "  build-all      Build frontend + Go binary"
        Write-Host "  dev            Build and run server"
        Write-Host "  dev-frontend   Start Vite dev server (http://localhost:5173)"
        Write-Host "  help           Show this help"
        Write-Host ""
        Write-Host "Options:" -ForegroundColor Yellow
        Write-Host "  --logfile [path]  Mirror output to a log file (default: output/build.log)"
        Write-Host "  --help            Show this help"
        Write-Host ""
        Write-Host "Examples:" -ForegroundColor Yellow
        Write-Host "  .\build.ps1 dev                         # Start backend"
        Write-Host "  .\build.ps1 dev-frontend                # Start frontend"
        Write-Host "  .\build.ps1 build-all                   # Production build"
        Write-Host "  .\build.ps1 dev --logfile               # Dev with log (default path)"
        Write-Host "  .\build.ps1 build --logfile build.log   # Build with custom log path"
        Write-Host ""
    }
}
