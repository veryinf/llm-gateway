#!/usr/bin/env python3
"""LLM Gateway LLM 代理接口测试脚本

测试 OpenAI 兼容 /v1/chat/completions 与 Anthropic 兼容 /anthropic/v1/messages
的流式与非流式响应。Bearer sk- 鉴权。

用法：
    python request.py                # 默认测试 OpenAI
    python request.py openai         # 测试 OpenAI 接口
    python request.py anthropic      # 测试 Anthropic 接口
    python request.py all            # 测试两种接口

环境：
    LLM_API_KEY  sk-xxx，默认 Bearer 密钥（也可通过 --api-key 传入）
    LLM_HOST     服务器地址，默认 http://localhost:3001
"""

import argparse
import json
import os
import sys
import urllib.error
import urllib.request

# ============================================================
# 用户配置（修改此处即可自定义测试，CLI 参数 / 环境变量可覆盖）
# ============================================================
DEFAULT_HOST: str = "http://localhost:3001"        # 服务器地址
DEFAULT_MODEL: str = "basic"                   # 测试模型（OpenAI / Anthropic 共用）
DEFAULT_API_KEY: str = "sk-c7d79944f064315b0ce5a04c3c1daef632b43401765eea13f5fd391f0c6732c3"                          # 默认 sk- 密钥（留空则强制走 --api-key 或 LLM_API_KEY）
PROMPT_NON_STREAM: str = "你好，请用一句话介绍自己"  # 非流式测试提示词
PROMPT_STREAM: str = "从1数到5"                    # 流式测试提示词

_USE_COLOR = True


def _set_color(enabled: bool) -> None:
    global _USE_COLOR
    _USE_COLOR = enabled


class Color:
    RESET = "\033[0m"
    RED = "\033[31m"
    GREEN = "\033[32m"
    YELLOW = "\033[33m"
    CYAN = "\033[36m"
    BOLD = "\033[1m"
    DIM = "\033[2m"

    @classmethod
    def wrap(cls, text: str, color: str) -> str:
        if not _USE_COLOR:
            return text
        return f"{color}{text}{cls.RESET}"


# ============================================================
# HTTP 封装
# ============================================================
def _post_json(url: str, body: dict, headers: dict, timeout: int = 60) -> tuple[int, str]:
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=payload, method="POST", headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            return resp.status, resp.read().decode("utf-8")
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8")
    except urllib.error.URLError as e:
        return -1, f"连接失败: {e.reason}"
    except TimeoutError:
        return -1, f"请求超时 ({timeout}s)"
    except Exception as e:
        return -1, f"请求异常: {e}"


def _post_stream(url: str, body: dict, headers: dict, on_event, timeout: int = 60) -> tuple[int, str]:
    """流式请求：on_event(json_dict) 由调用方决定何时停止（返回 False 可中断）。"""
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=payload, method="POST", headers=headers)
    resp = None
    try:
        resp = urllib.request.urlopen(req, timeout=timeout)
        for raw_line in resp:
            try:
                line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")
            except Exception:
                continue
            if not line.startswith("data: "):
                continue
            data = line[len("data: "):]
            if data == "[DONE]":
                break
            try:
                obj = json.loads(data)
            except json.JSONDecodeError:
                continue
            if not on_event(obj):
                break
        return 200, ""
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode("utf-8")
    except urllib.error.URLError as e:
        return -1, f"连接失败: {e.reason}"
    except TimeoutError:
        return -1, f"请求超时 ({timeout}s)"
    except Exception as e:
        return -1, f"请求异常: {e}"
    finally:
        if resp is not None:
            try:
                resp.close()
            except Exception:
                pass


# ============================================================
# OpenAI 测试
# ============================================================
def test_openai(host: str, api_key: str, model: str) -> None:
    print(Color.wrap("=== OpenAI API 测试 ===", Color.CYAN))
    print()

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}",
    }

    # 1. 非流式
    print(Color.wrap("1. Chat Completions（非流式）", Color.YELLOW))
    body = {
        "model": model,
        "messages": [{"role": "user", "content": PROMPT_NON_STREAM}],
        "stream": False,
    }
    status, text = _post_json(f"{host}/v1/chat/completions", body, headers)
    if status == 200:
        try:
            result = json.loads(text)
            content = result.get("choices", [{}])[0].get("message", {}).get("content", "")
            usage = result.get("usage", {}) or {}
            print(Color.wrap(f"   模型: {result.get('model', model)}", Color.GREEN))
            print(Color.wrap(f"   回复: {content}", Color.GREEN))
            print(Color.wrap(
                f"   Token: 提示={usage.get('prompt_tokens')}, "
                f"补全={usage.get('completion_tokens')}, "
                f"总计={usage.get('total_tokens')}",
                Color.DIM,
            ))
        except json.JSONDecodeError:
            print(Color.wrap(f"   解析失败: {text[:200]}", Color.RED))
    else:
        print(Color.wrap(f"   请求失败 [{status}]: {text[:200]}", Color.RED))
    print()

    # 2. 流式
    print(Color.wrap("2. Chat Completions（流式）", Color.YELLOW))
    body["messages"] = [{"role": "user", "content": PROMPT_STREAM}]
    body["stream"] = True
    print("   流式响应: ", end="", flush=True)

    def on_event(obj):
        delta = obj.get("choices", [{}])[0].get("delta", {}) or {}
        content = delta.get("content")
        if content:
            print(Color.wrap(content, Color.GREEN), end="", flush=True)
        return True

    status, text = _post_stream(f"{host}/v1/chat/completions", body, headers, on_event)
    print()
    if status != 200:
        print(Color.wrap(f"   流式请求失败 [{status}]: {text[:200]}", Color.RED))
    print()


# ============================================================
# Anthropic 测试
# ============================================================
def test_anthropic(host: str, api_key: str, model: str) -> None:
    print(Color.wrap("=== Anthropic API 测试 ===", Color.CYAN))
    print()

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}",
        "anthropic-version": "2023-06-01",
    }

    # 1. 非流式
    print(Color.wrap("1. Messages（非流式）", Color.YELLOW))
    body = {
        "model": model,
        "max_tokens": 1024,
        "messages": [{"role": "user", "content": PROMPT_NON_STREAM}],
    }
    status, text = _post_json(f"{host}/anthropic/v1/messages", body, headers)
    if status == 200:
        try:
            result = json.loads(text)
            content_arr = result.get("content") or []
            text_content = next((c.get("text", "") for c in content_arr if c.get("type") == "text"), "")
            usage = result.get("usage", {}) or {}
            print(Color.wrap(f"   模型: {result.get('model', model)}", Color.GREEN))
            print(Color.wrap(f"   回复: {text_content}", Color.GREEN))
            print(Color.wrap(
                f"   Token: 输入={usage.get('input_tokens')}, "
                f"输出={usage.get('output_tokens')}",
                Color.DIM,
            ))
        except json.JSONDecodeError:
            print(Color.wrap(f"   解析失败: {text[:200]}", Color.RED))
    else:
        print(Color.wrap(f"   请求失败 [{status}]: {text[:200]}", Color.RED))
    print()

    # 2. 流式
    print(Color.wrap("2. Messages（流式）", Color.YELLOW))
    body["messages"] = [{"role": "user", "content": PROMPT_STREAM}]
    body["stream"] = True
    print("   流式响应: ", end="", flush=True)

    def on_event(obj):
        if obj.get("type") == "content_block_delta":
            delta = obj.get("delta", {}) or {}
            chunk = delta.get("text")
            if chunk:
                print(Color.wrap(chunk, Color.GREEN), end="", flush=True)
        return True

    status, text = _post_stream(f"{host}/anthropic/v1/messages", body, headers, on_event)
    print()
    if status != 200:
        print(Color.wrap(f"   流式请求失败 [{status}]: {text[:200]}", Color.RED))
    print()


# ============================================================
# CLI
# ============================================================
def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="request.py",
        description="LLM Gateway LLM 代理接口测试（Bearer sk- Key）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "示例：\n"
            "  python request.py                # 默认测试 OpenAI 接口\n"
            "  python request.py openai         # 测试 OpenAI 接口\n"
            "  python request.py anthropic      # 测试 Anthropic 接口\n"
            "  python request.py all            # 测试两种接口\n"
        ),
    )
    parser.add_argument(
        "api",
        nargs="?",
        default="openai",
        choices=["openai", "anthropic", "all"],
        help="测试接口（默认 openai）",
    )
    parser.add_argument("--host", default=os.environ.get("LLM_HOST", DEFAULT_HOST), help=f"服务器地址，默认 {DEFAULT_HOST}")
    parser.add_argument(
        "--api-key",
        help="Bearer sk- API Key（默认从 LLM_API_KEY 环境变量读取）",
    )
    parser.add_argument("--model", default=DEFAULT_MODEL, help=f"测试模型（默认 {DEFAULT_MODEL}）")
    parser.add_argument("--timeout", type=int, default=60, help="超时秒数，默认 60")
    parser.add_argument("--no-color", action="store_true", help="禁用彩色输出")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    _set_color(not args.no_color)

    api_key = args.api_key or os.environ.get("LLM_API_KEY") or os.environ.get("API_KEY") or DEFAULT_API_KEY
    if not api_key:
        print(Color.wrap("✗ 缺少 API Key", Color.RED), file=sys.stderr)
        print("  请通过 --api-key 参数、LLM_API_KEY 环境变量或 DEFAULT_API_KEY 配置传入 sk- 密钥", file=sys.stderr)
        return 2

    host = args.host.rstrip("/")
    model = args.model

    if args.api in ("openai", "all"):
        test_openai(host, api_key, model)

    if args.api in ("anthropic", "all"):
        if args.api == "all":
            print(Color.wrap("=" * 50, Color.DIM))
            print()
        test_anthropic(host, api_key, model)

    print(Color.wrap("=== 测试完成 ===", Color.CYAN))
    return 0


if __name__ == "__main__":
    sys.exit(main())