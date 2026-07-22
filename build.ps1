# LLM Gateway Build Script
# Usage: .\build.ps1 <command> [options]
# Commands: build, build-all, dev, debug, dev-frontend, package, help

$BinaryName = "lgw.exe"
$BuildDir = "out"
$DataDir = "db"
$WebDir = "web"

# Parse $args manually (PowerShell doesn't support -- prefix in param)
$Command = "help"
$LogBoth = $false
$PushImage = $false
$help = $false

$i = 0
while ($i -lt $args.Count) {
    switch ($args[$i]) {
        '--logfile' { $LogBoth = $true }
        '--push'    { $PushImage = $true }
        '--help'    { $help = $true }
        default {
            if ($Command -eq "help") { $Command = $args[$i] }
        }
    }
    $i++
}
if ($help) { $Command = "help" }

function Write-Tee {
    param([string]$msg, [string]$color)
    if ($color) { Write-Host $msg -ForegroundColor $color } else { Write-Host $msg }
}

function Run {
    param([string]$cmd)
    Invoke-Expression $cmd
}

function Write-Step { param([string]$msg) Write-Tee "==> $msg" "Cyan" }
function Write-OK { param([string]$msg) Write-Tee "√ $msg" "Green" }
function Write-Err { param([string]$msg) Write-Tee "× $msg" "Red" }

$BuildVersion = if (Test-Path env:CI_COMMIT_TAG) { $env:CI_COMMIT_TAG } else { "dev" }
$BuildTime = Get-Date -Format "2006-01-02 15:04:05"
$IsProdBuild = $Command -eq "build" -or $Command -eq "build-all"
$BuildEnv = if ($IsProdBuild) { "production" } else { "development" }

# 变量注入（生产/开发共用）
$VarFlags = "-X 'llm-gateway/internal/core.BuildEnv=$BuildEnv' -X 'llm-gateway/internal/core.BuildTime=$BuildTime' -X 'llm-gateway/internal/core.BuildVersion=$BuildVersion'"

# 生产 ldflags：剥离符号表 (-s) + DWARF 调试信息 (-w)
$ProdLdFlags = "-s -w $VarFlags"

# 开发 ldflags：保留符号，便于 panic 堆栈定位、pprof 函数名、dlv 调试
$DevLdFlags = $VarFlags

switch ($Command) {
    "build" {
        Write-Step "Building $BinaryName (production, stripped)..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        Run "go build -ldflags=""$ProdLdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/"
        if ($LASTEXITCODE -eq 0) {
            Write-OK "Build success: $BuildDir/$BinaryName"
        } else {
            Write-Err "Build failed"
            exit $LASTEXITCODE
        }
    }

    "dev" {
        $BinaryName = "lgw-dev.exe"
        New-Item -ItemType Directory -Force -Path $DataDir, $BuildDir | Out-Null
        Write-Step "Building (dev, symbols kept) and starting server..."
        Run "go build -ldflags=""$DevLdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/"
        if ($LASTEXITCODE -ne 0) { Write-Err "Build failed"; exit $LASTEXITCODE }
        Write-OK "Build done, starting..."
        $startArgs = @()
        if ($LogBoth) { $startArgs += "--log", "both" }
        & "$BuildDir/$BinaryName" @startArgs
    }

    "debug" {
        $BinaryName = "lgw-dev.exe"
        New-Item -ItemType Directory -Force -Path $DataDir, $BuildDir | Out-Null
        Write-Step "Building (debug, symbols kept) and starting with Delve..."
        Run "go build -ldflags=""$DevLdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/"
        if ($LASTEXITCODE -ne 0) { Write-Err "Build failed"; exit $LASTEXITCODE }
        Write-OK "Build done, starting dlv (IDE attach on :2345)..."
        $startArgs = @("exec", "$BuildDir/$BinaryName", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient", "--")
        if ($LogBoth) { $startArgs += "--log", "both" }
        & dlv @startArgs
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
        Write-Step "Building $BinaryName (production, stripped, with embedded frontend)..."
        New-Item -ItemType Directory -Force -Path $BuildDir | Out-Null
        Run "go build -ldflags=""$ProdLdFlags"" -o ""$BuildDir/$BinaryName"" ./cmd/"
        if ($LASTEXITCODE -eq 0) { Write-OK "Build all → $BuildDir/$BinaryName (standalone, frontend embedded)" }
    }

    "dev-frontend" {
        Write-Step "Starting frontend dev server..."
        Push-Location $WebDir
        try { pnpm dev } finally { Pop-Location }
    }

    "package" {
        $Wslc = "C:\Program Files\WSL\wslc.exe"
        if (!(Test-Path $Wslc)) {
            Write-Err "wslc not found at $Wslc (WSL container CLI is required for image packaging)"
            exit 1
        }
        $ImageName = "git.chuangyun.work/archive/llm-gateway:latest"
        $BuildTime = Get-Date -Format "2006-01-02 15:04:05"

        Write-Step "Building image $ImageName via wslc (frontend + Go binary multi-stage)..."
        $buildArgs = @(
            "build",
            "-t", $ImageName,
            "-f", "Dockerfile",
            "--build-arg", "VERSION=$BuildVersion",
            "--build-arg", "BUILD_TIME=$BuildTime",
            "--build-arg", "BUILD_ENV=production",
            "."
        )
        & $Wslc @buildArgs
        if ($LASTEXITCODE -ne 0) { Write-Err "wslc build failed"; exit $LASTEXITCODE }
        Write-OK "Image built: $ImageName"

        if ($PushImage) {
            Write-Step "Pushing $ImageName to registry..."
            & $Wslc push $ImageName
            if ($LASTEXITCODE -ne 0) { Write-Err "wslc push failed"; exit $LASTEXITCODE }
            Write-OK "Image pushed"
        } else {
            Write-Host ""
            Write-Host "  Tip: pass --push to also upload to the registry" -ForegroundColor DarkGray
        }
    }

    default {
        Write-Host ""
        Write-Host "LLM Gateway Build Script" -ForegroundColor Cyan
        Write-Host "========================" -ForegroundColor Cyan
        Write-Host ""
        Write-Host "Usage: .\build.ps1 <command> [options]"
        Write-Host ""
        Write-Host "Commands:" -ForegroundColor Yellow
        Write-Host "  build          Build Go binary -> output/"
        Write-Host "  build-all      Build frontend + Go binary (frontend embedded)"
        Write-Host "  dev            Build and run server (supports --logfile)"
        Write-Host "  debug          Build and start with Delve debugger on :2345 (supports --logfile)"
        Write-Host "  dev-frontend   Start Vite dev server (http://localhost:3000)"
        Write-Host "  package        Build container image via wslc (supports --push)"
        Write-Host "  help           Show this help"
        Write-Host ""
        Write-Host "Options:" -ForegroundColor Yellow
        Write-Host "  --logfile        Pass --log both to the binary (output to console + file)"
        Write-Host "  --push           (package) also push the built image to the registry"
        Write-Host "  --help           Show this help"
        Write-Host ""
        Write-Host "Examples:" -ForegroundColor Yellow
        Write-Host "  .\build.ps1 dev                         # Start backend"
        Write-Host "  .\build.ps1 dev --logfile               # Dev with console + file logging"
        Write-Host "  .\build.ps1 debug                       # Build + start Delve (IDE attach on :2345)"
        Write-Host "  .\build.ps1 debug --logfile             # Debug with console + file logging"
        Write-Host "  .\build.ps1 package                     # Build image git.chuangyun.work/archive/llm-gateway:latest"
        Write-Host "  .\build.ps1 package --push              # Build + push image to registry"
        Write-Host ""
    }
}
