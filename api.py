#!/usr/bin/env python3
"""LLM Gateway 通用 API 测试脚本

通过命令行参数调用后端任意 Provider/UserModel 等 CRUD 接口。

用法：
    python api.py <endpoint> [--data <json>] [--host <url>] [--no-color]

示例：
    python api.py providers/search --data '{"kw":"openai"}'
    python api.py providers/fetch --data '{"providerId":1}'
    python api.py providers/add --data '{"title":"t","baseUrl":"https://api.openai.com","apiKey":"sk-test","supportOpenai":true,"models":["gpt-4o"]}'
    python api.py providers/update --data '{"providerId":1,"isActive":false}'
    python api.py providers/remove --data '{"providerId":1}'

环境：
    默认从 web/.env 读取 VITE_API_KEY / VITE_API_SECRET
    可通过环境变量 API_KEY / API_SECRET 覆盖
    默认服务器地址 http://localhost:3001
"""

import argparse
import hashlib
import json
import os
import sys
import time
from pathlib import Path
from urllib import request as urlrequest
from urllib.error import HTTPError, URLError

# ============================================================
# 常量
# ============================================================
DEFAULT_HOST = "http://localhost:3001"
API_PREFIX = "/api"
SCRIPT_DIR = Path(__file__).resolve().parent
ENV_FILE = SCRIPT_DIR / "web" / ".env"

# ANSI 颜色
_USE_COLOR = True


def _set_color_enabled(enabled: bool) -> None:
    global _USE_COLOR
    _USE_COLOR = enabled


class Color:
    RESET = "\033[0m"
    RED = "\033[31m"
    GREEN = "\033[32m"
    YELLOW = "\033[33m"
    BLUE = "\033[34m"
    CYAN = "\033[36m"
    BOLD = "\033[1m"
    DIM = "\033[2m"

    @classmethod
    def wrap(cls, text: str, color: str) -> str:
        if not _USE_COLOR:
            return text
        return f"{color}{text}{cls.RESET}"


# ============================================================
# 配置加载
# ============================================================
def load_credentials() -> tuple[str, str]:
    """从 web/.env 加载 AK/SK，环境变量优先。"""
    api_key: str | None = None
    api_secret: str | None = None

    if ENV_FILE.exists():
        for line in ENV_FILE.read_text(encoding="utf-8").splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            k = k.strip()
            v = v.strip().strip('"').strip("'")
            if k == "VITE_API_KEY":
                api_key = v
            elif k == "VITE_API_SECRET":
                api_secret = v

    api_key = os.environ.get("API_KEY", api_key)
    api_secret = os.environ.get("API_SECRET", api_secret)

    if not api_key or not api_secret:
        print(Color.wrap("✗ 缺少 AK/SK 配置", Color.RED), file=sys.stderr)
        print(f"  请检查 {ENV_FILE} 或设置环境变量 API_KEY / API_SECRET", file=sys.stderr)
        sys.exit(2)

    return api_key, api_secret


# ============================================================
# 签名
# ============================================================
def make_signature(timestamp: str, secret: str) -> str:
    """MD5(timestamp + secret) - 与前端 web/src/lib/request.ts 一致"""
    return hashlib.md5(f"{timestamp}{secret}".encode("utf-8")).hexdigest()


# ============================================================
# HTTP 调用
# ============================================================
def call_api(host: str, endpoint: str, data: dict, api_key: str, api_secret: str, timeout: int = 30) -> dict:
    """调用 API，返回 {status, body}。"""
    timestamp = str(int(time.time()))
    signature = make_signature(timestamp, api_secret)

    url = f"{host.rstrip('/')}{API_PREFIX}/{endpoint.lstrip('/')}"
    payload = json.dumps(data, ensure_ascii=False).encode("utf-8")

    req = urlrequest.Request(
        url,
        data=payload,
        method="POST",
        headers={
            "Content-Type": "application/json",
            "X-Api-Key": api_key,
            "X-Api-Time": timestamp,
            "X-Api-Signature": signature,
        },
    )

    try:
        with urlrequest.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            return {"status": resp.status, "body": _safe_json(raw)}
    except HTTPError as e:
        raw = e.read().decode("utf-8")
        return {"status": e.code, "body": _safe_json(raw)}
    except URLError as e:
        return {"status": -1, "body": {"errCode": -1, "errMsg": f"连接失败: {e.reason}"}}
    except TimeoutError:
        return {"status": -1, "body": {"errCode": -1, "errMsg": f"请求超时 ({timeout}s)"}}


def _safe_json(text: str) -> dict:
    if not text:
        return {}
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return {"errCode": -1, "errMsg": f"非 JSON 响应: {text[:200]}"}


# ============================================================
# 输出
# ============================================================
def print_request(host: str, endpoint: str, data: dict) -> None:
    url = f"{host.rstrip('/')}{API_PREFIX}/{endpoint.lstrip('/')}"
    print(f"{Color.wrap('▶ 请求', Color.CYAN)} {Color.wrap(f'POST {url}', Color.BOLD + Color.CYAN)}")
    if data:
        print(f"  {Color.wrap('Body: ' + json.dumps(data, ensure_ascii=False), Color.DIM)}")
    else:
        print(f"  {Color.wrap('Body: {}', Color.DIM)}")


def print_response(result: dict) -> None:
    status = result["status"]
    body = result.get("body", {}) or {}

    if 200 <= status < 300:
        head_color = Color.GREEN
        head_label = "✓ 响应"
    elif status == -1:
        head_color = Color.YELLOW
        head_label = "⚠ 网络"
    else:
        head_color = Color.RED
        head_label = "✗ 响应"

    print(f"{head_color}{head_label}{Color.RESET} HTTP {status}")

    err_code = body.get("errCode")
    if err_code == 0:
        print(f"  {Color.wrap(f'errCode: {err_code} (成功)', Color.GREEN)}")
    elif err_code is not None:
        msg = body.get("errMsg", "")
        print(f"  {Color.wrap(f'errCode: {err_code}, errMsg: {msg}', Color.RED)}")
    else:
        print(f"  {Color.wrap(f'{json.dumps(body, ensure_ascii=False)}', Color.RED)}")

    pretty = json.dumps(body, ensure_ascii=False, indent=2)
    for line in pretty.splitlines():
        print(f"  {Color.wrap(line, Color.DIM)}")


# ============================================================
# CLI
# ============================================================
def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="api.py",
        description="LLM Gateway 通用 API 调用工具（AKSK 签名）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "示例：\n"
            '  python api.py providers/search --data \'{"kw":"openai"}\'\n'
            '  python api.py providers/fetch --data \'{"providerId":1}\'\n'
            '  python api.py providers/add --data \'{"title":"t","baseUrl":"https://api.openai.com","apiKey":"sk-test","supportOpenai":true}\'\n'
            '  python api.py providers/update --data \'{"providerId":1,"isActive":false}\'\n'
            '  python api.py providers/remove --data \'{"providerId":1}\'\n'
        ),
    )
    parser.add_argument("endpoint", help="API 路径（不带 /api 前缀），如 providers/search")
    parser.add_argument("--data", "-d", default="", help="JSON 请求体字符串")
    parser.add_argument("--host", default=DEFAULT_HOST, help=f"服务器地址，默认 {DEFAULT_HOST}")
    parser.add_argument("--timeout", type=int, default=30, help="超时秒数，默认 30")
    parser.add_argument("--no-color", action="store_true", help="禁用彩色输出")
    parser.add_argument("--show-headers", action="store_true", help="打印请求头（调试用）")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    _set_color_enabled(not args.no_color)

    # 解析 data
    if args.data.strip():
        try:
            data = json.loads(args.data)
        except json.JSONDecodeError as e:
            print(Color.wrap(f"✗ --data 不是合法 JSON: {e}", Color.RED), file=sys.stderr)
            return 2
    else:
        data = {}

    # 加载凭证
    api_key, api_secret = load_credentials()

    # 打印请求
    print_request(args.host, args.endpoint, data)

    if args.show_headers:
        ts = str(int(time.time()))
        sig = make_signature(ts, api_secret)
        print(f"  {Color.wrap(f'X-Api-Key: {api_key}', Color.DIM)}")
        print(f"  {Color.wrap(f'X-Api-Time: {ts}', Color.DIM)}")
        print(f"  {Color.wrap(f'X-Api-Signature: {sig}', Color.DIM)}")

    # 调用
    result = call_api(args.host, args.endpoint, data, api_key, api_secret, timeout=args.timeout)

    # 打印响应
    print()
    print_response(result)

    # 退出码
    body = result.get("body", {}) or {}
    if result["status"] == 200 and body.get("errCode") == 0:
        return 0
    return 1


if __name__ == "__main__":
    sys.exit(main())