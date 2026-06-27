# LLM Gateway API 测试脚本
# 使用方法:
#   .\request.ps1                # 默认测试 OpenAI 接口
#   .\request.ps1 openai         # 测试 OpenAI 接口
#   .\request.ps1 anthropic      # 测试 Anthropic 接口
#   .\request.ps1 all            # 测试两种接口

param(
    [ValidateSet("openai", "anthropic", "all")]
    [string]$Api = "openai"
)

$BaseUrl = "http://localhost:3001"
$ApiKey = "sk-c7d79944f064315b0ce5a04c3c1daef632b43401765eea13f5fd391f0c6732c3"

function Test-OpenAI {
    Write-Host "=== OpenAI API 测试 ===" -ForegroundColor Cyan
    Write-Host ""

    $apiHeaders = @{
        Authorization = "Bearer $ApiKey"
    }

    # 1. 非流式请求
    Write-Host "1. Chat Completions（非流式）" -ForegroundColor Yellow
    try {
        $chatBody = @{
            model = "mimo-v2.5"
            messages = @(
                @{
                    role = "user"
                    content = "你好，请用一句话介绍自己"
                }
            )
            stream = $false
        } | ConvertTo-Json -Depth 10

        $chatResult = Invoke-RestMethod -Uri "$BaseUrl/v1/chat/completions" -Method Post -Body $chatBody -ContentType "application/json" -Headers $apiHeaders
        Write-Host "   模型: $($chatResult.model)" -ForegroundColor Green
        Write-Host "   回复: $($chatResult.choices[0].message.content)" -ForegroundColor Green
        Write-Host "   Token: 提示=$($chatResult.usage.prompt_tokens), 补全=$($chatResult.usage.completion_tokens), 总计=$($chatResult.usage.total_tokens)" -ForegroundColor Gray
    } catch {
        Write-Host "   请求失败: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""

    # 2. 流式请求
    Write-Host "2. Chat Completions（流式）" -ForegroundColor Yellow
    try {
        $streamBody = @{
            model = "mimo-v2.5"
            messages = @(
                @{
                    role = "user"
                    content = "从1数到5"
                }
            )
            stream = $true
        } | ConvertTo-Json -Depth 10

        Write-Host "   流式响应: " -NoNewline -ForegroundColor Gray

        Add-Type -AssemblyName System.Net.Http
        $httpClient = [System.Net.Http.HttpClient]::new()
        $httpClient.DefaultRequestHeaders.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $ApiKey)
        $httpClient.DefaultRequestHeaders.UserAgent.ParseAdd("PowerShell/LLM-Gateway-Test")

        $content = [System.Net.Http.StringContent]::new($streamBody, [System.Text.Encoding]::UTF8, "application/json")
        $response = $httpClient.PostAsync("$BaseUrl/v1/chat/completions", $content).Result
        $stream = $response.Content.ReadAsStreamAsync().Result
        $reader = [System.IO.StreamReader]::new($stream)

        while (-not $reader.EndOfStream) {
            $line = $reader.ReadLine()
            if ($line -match "^data: (.+)$") {
                $data = $Matches[1]
                if ($data -ne "[DONE]") {
                    try {
                        $json = $data | ConvertFrom-Json
                        if ($json.choices[0].delta.content) {
                            Write-Host $json.choices[0].delta.content -NoNewline -ForegroundColor Green
                        }
                    } catch {}
                }
            }
        }
        Write-Host ""
        $reader.Close()
        $stream.Close()
        $httpClient.Dispose()
    } catch {
        Write-Host "   流式请求失败: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""
}

function Test-Anthropic {
    Write-Host "=== Anthropic API 测试 ===" -ForegroundColor Cyan
    Write-Host ""

    $apiHeaders = @{
        Authorization = "Bearer $ApiKey"
        "anthropic-version" = "2023-06-01"
    }

    # 1. 非流式请求
    Write-Host "1. Messages（非流式）" -ForegroundColor Yellow
    try {
        $chatBody = @{
            model = "mimo-v2.5"
            max_tokens = 1024
            messages = @(
                @{
                    role = "user"
                    content = "你好，请用一句话介绍自己"
                }
            )
        } | ConvertTo-Json -Depth 10

        $chatResult = Invoke-RestMethod -Uri "$BaseUrl/anthropic/v1/messages" -Method Post -Body $chatBody -ContentType "application/json" -Headers $apiHeaders
        Write-Host "   模型: $($chatResult.model)" -ForegroundColor Green
        Write-Host "   回复: $($chatResult.content[0].text)" -ForegroundColor Green
        Write-Host "   Token: 输入=$($chatResult.usage.input_tokens), 输出=$($chatResult.usage.output_tokens)" -ForegroundColor Gray
    } catch {
        Write-Host "   请求失败: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""

    # 2. 流式请求
    Write-Host "2. Messages（流式）" -ForegroundColor Yellow
    try {
        $streamBody = @{
            model = "mimo-v2.5"
            max_tokens = 1024
            messages = @(
                @{
                    role = "user"
                    content = "从1数到5"
                }
            )
            stream = $true
        } | ConvertTo-Json -Depth 10

        Write-Host "   流式响应: " -NoNewline -ForegroundColor Gray

        Add-Type -AssemblyName System.Net.Http
        $httpClient = [System.Net.Http.HttpClient]::new()
        $httpClient.DefaultRequestHeaders.Authorization = [System.Net.Http.Headers.AuthenticationHeaderValue]::new("Bearer", $ApiKey)
        $httpClient.DefaultRequestHeaders.UserAgent.ParseAdd("PowerShell/LLM-Gateway-Test")
        $httpClient.DefaultRequestHeaders.Add("anthropic-version", "2023-06-01")

        $content = [System.Net.Http.StringContent]::new($streamBody, [System.Text.Encoding]::UTF8, "application/json")
        $response = $httpClient.PostAsync("$BaseUrl/anthropic/v1/messages", $content).Result
        $stream = $response.Content.ReadAsStreamAsync().Result
        $reader = [System.IO.StreamReader]::new($stream)

        while (-not $reader.EndOfStream) {
            $line = $reader.ReadLine()
            if ($line -match "^data: (.+)$") {
                $data = $Matches[1]
                try {
                    $json = $data | ConvertFrom-Json
                    if ($json.type -eq "content_block_delta" -and $json.delta.text) {
                        Write-Host $json.delta.text -NoNewline -ForegroundColor Green
                    }
                } catch {}
            }
        }
        Write-Host ""
        $reader.Close()
        $stream.Close()
        $httpClient.Dispose()
    } catch {
        Write-Host "   流式请求失败: $($_.Exception.Message)" -ForegroundColor Red
    }
    Write-Host ""
}

# 执行测试
switch ($Api) {
    "openai"    { Test-OpenAI }
    "anthropic" { Test-Anthropic }
    "all"       {
        Test-OpenAI
        Write-Host ("=" * 50) -ForegroundColor DarkGray
        Write-Host ""
        Test-Anthropic
    }
}

Write-Host "=== 测试完成 ===" -ForegroundColor Cyan
