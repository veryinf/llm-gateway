#!/usr/bin/env python3
"""流式响应逐 frame 测试脚本

验证流式响应是否逐 chunk 到达（而非一次性到达），输出每个 chunk 的时间戳和间隔。

用法：
    python stream_test.py                # 默认测试 OpenAI
    python stream_test.py openai         # 测试 OpenAI 接口
    python stream_test.py anthropic      # 测试 Anthropic 接口
    python stream_test.py all            # 测试两种接口

环境：
    LLM_API_KEY  sk-xxx
    LLM_HOST     服务器地址，默认 http://localhost:3001
"""

import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request

# ============================================================
# 配置
# ============================================================
DEFAULT_HOST = "http://localhost:3001"
DEFAULT_MODEL = "basic"
DEFAULT_API_KEY = "sk-c7d79944f064315b0ce5a04c3c1daef632b43401765eea13f5fd391f0c6732c3"
PROMPT_STREAM = "从1数到5，每个数字后面换行"  # 期望产生多个 chunk 的提示词

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
# 流式测试核心
# ============================================================
def test_stream_openai(host: str, api_key: str, model: str) -> None:
    print(Color.wrap("=== OpenAI 流式逐 frame 测试 ===", Color.CYAN))
    print()

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}",
    }
    body = {
        "model": model,
        "messages": [{"role": "user", "content": PROMPT_STREAM}],
        "stream": True,
    }

    url = f"{host}/v1/chat/completions"
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=payload, method="POST", headers=headers)

    chunks = []
    start_time = None
    first_chunk_time = None
    last_chunk_time = None

    try:
        resp = urllib.request.urlopen(req, timeout=60)
        print(Color.wrap("连接成功，等待 chunk...", Color.YELLOW))
        print()
        print(f"{'#':>3}  {'间隔(ms)':>10}  {'耗时(ms)':>10}  {'内容'}")
        print("-" * 70)

        for raw_line in resp:
            now = time.time()
            if start_time is None:
                start_time = now
                first_chunk_time = now
                last_chunk_time = now

            try:
                line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")
            except Exception:
                continue

            if not line.startswith("data: "):
                continue

            data = line[len("data: "):]
            if data == "[DONE]":
                chunks.append({"type": "done", "data": "[DONE]", "time": now})
                elapsed_ms = (now - start_time) * 1000
                interval_ms = (now - last_chunk_time) * 1000
                print(f"{len(chunks):>3}  {interval_ms:>10.1f}  {elapsed_ms:>10.1f}  {Color.wrap('[DONE]', Color.DIM)}")
                break

            try:
                obj = json.loads(data)
            except json.JSONDecodeError:
                continue

            choices = obj.get("choices") or []
            if not choices:
                continue
            delta = choices[0].get("delta", {}) or {}
            content = delta.get("content") or delta.get("reasoning_content") or ""
            chunk_type = "reasoning" if delta.get("reasoning_content") else ("content" if content else "other")

            now = time.time()
            elapsed_ms = (now - start_time) * 1000
            interval_ms = (now - last_chunk_time) * 1000
            last_chunk_time = now

            chunks.append({"type": chunk_type, "data": content, "time": now})

            display = repr(content) if content else Color.wrap("(空chunk)", Color.DIM)
            print(f"{len(chunks):>3}  {interval_ms:>10.1f}  {elapsed_ms:>10.1f}  {display}")

        resp.close()

    except urllib.error.HTTPError as e:
        print(Color.wrap(f"请求失败 [{e.code}]: {e.read().decode('utf-8')[:200]}", Color.RED))
        return
    except Exception as e:
        print(Color.wrap(f"异常: {e}", Color.RED))
        return

    # 汇总
    print()
    print(Color.wrap("--- 汇总 ---", Color.CYAN))
    total_chunks = len(chunks)
    total_time = (chunks[-1]["time"] - chunks[0]["time"]) * 1000 if total_chunks > 1 else 0
    intervals = []
    for i in range(1, len(chunks)):
        intervals.append((chunks[i]["time"] - chunks[i-1]["time"]) * 1000)

    print(f"  总 chunk 数: {total_chunks}")
    print(f"  总耗时: {total_time:.1f} ms")
    if intervals:
        avg_interval = sum(intervals) / len(intervals)
        min_interval = min(intervals)
        max_interval = max(intervals)
        print(f"  平均间隔: {avg_interval:.1f} ms")
        print(f"  最小间隔: {min_interval:.1f} ms")
        print(f"  最大间隔: {max_interval:.1f} ms")

        # 判断是否是逐 frame 还是批量
        # 如果所有间隔都 < 5ms，可能是批量到达
        batch_threshold = 5.0  # ms
        all_fast = all(i < batch_threshold for i in intervals)
        if all_fast and len(intervals) > 1:
            print()
            print(Color.wrap("  ⚠ 警告: 所有 chunk 间隔均 < 5ms，疑似批量到达而非逐 frame！", Color.YELLOW))
        elif len(intervals) > 1:
            print()
            print(Color.wrap("  ✓ chunk 间隔有明显差异，符合逐 frame 特征", Color.GREEN))
    print()


def test_stream_anthropic(host: str, api_key: str, model: str) -> None:
    print(Color.wrap("=== Anthropic 流式逐 frame 测试 ===", Color.CYAN))
    print()

    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {api_key}",
        "anthropic-version": "2023-06-01",
    }
    body = {
        "model": model,
        "max_tokens": 1024,
        "messages": [{"role": "user", "content": PROMPT_STREAM}],
        "stream": True,
    }

    url = f"{host}/anthropic/v1/messages"
    payload = json.dumps(body, ensure_ascii=False).encode("utf-8")
    req = urllib.request.Request(url, data=payload, method="POST", headers=headers)

    chunks = []
    start_time = None
    last_chunk_time = None

    try:
        resp = urllib.request.urlopen(req, timeout=60)
        print(Color.wrap("连接成功，等待 chunk...", Color.YELLOW))
        print()
        print(f"{'#':>3}  {'间隔(ms)':>10}  {'耗时(ms)':>10}  {'类型':<25}  {'内容'}")
        print("-" * 90)

        for raw_line in resp:
            now = time.time()
            if start_time is None:
                start_time = now
                last_chunk_time = now

            try:
                line = raw_line.decode("utf-8", errors="replace").rstrip("\r\n")
            except Exception:
                continue

            if not line.startswith("data: "):
                continue

            data = line[len("data: "):]

            try:
                obj = json.loads(data)
            except json.JSONDecodeError:
                continue

            event_type = obj.get("type", "unknown")
            now = time.time()
            elapsed_ms = (now - start_time) * 1000
            interval_ms = (now - last_chunk_time) * 1000
            last_chunk_time = now

            display_content = ""
            if event_type == "content_block_delta":
                delta = obj.get("delta", {})
                display_content = delta.get("text") or delta.get("thinking") or ""
            elif event_type == "message_delta":
                display_content = f"stop_reason={obj.get('delta', {}).get('stop_reason', '')}"
            elif event_type == "message_stop":
                display_content = Color.wrap("(流结束)", Color.DIM)

            chunks.append({"type": event_type, "data": display_content, "time": now})

            display = repr(display_content) if display_content else Color.wrap("(空)", Color.DIM)
            print(f"{len(chunks):>3}  {interval_ms:>10.1f}  {elapsed_ms:>10.1f}  {event_type:<25}  {display}")

            if event_type == "message_stop":
                break

        resp.close()

    except urllib.error.HTTPError as e:
        print(Color.wrap(f"请求失败 [{e.code}]: {e.read().decode('utf-8')[:200]}", Color.RED))
        return
    except Exception as e:
        print(Color.wrap(f"异常: {e}", Color.RED))
        return

    # 汇总
    print()
    print(Color.wrap("--- 汇总 ---", Color.CYAN))
    total_chunks = len(chunks)
    total_time = (chunks[-1]["time"] - chunks[0]["time"]) * 1000 if total_chunks > 1 else 0
    intervals = []
    for i in range(1, len(chunks)):
        intervals.append((chunks[i]["time"] - chunks[i-1]["time"]) * 1000)

    print(f"  总 event 数: {total_chunks}")
    print(f"  总耗时: {total_time:.1f} ms")
    if intervals:
        avg_interval = sum(intervals) / len(intervals)
        min_interval = min(intervals)
        max_interval = max(intervals)
        print(f"  平均间隔: {avg_interval:.1f} ms")
        print(f"  最小间隔: {min_interval:.1f} ms")
        print(f"  最大间隔: {max_interval:.1f} ms")

        batch_threshold = 5.0
        all_fast = all(i < batch_threshold for i in intervals)
        if all_fast and len(intervals) > 1:
            print()
            print(Color.wrap("  ⚠ 警告: 所有 event 间隔均 < 5ms，疑似批量到达而非逐 frame！", Color.YELLOW))
        elif len(intervals) > 1:
            print()
            print(Color.wrap("  ✓ event 间隔有明显差异，符合逐 frame 特征", Color.GREEN))
    print()


# ============================================================
# CLI
# ============================================================
def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        prog="stream_test.py",
        description="LLM Gateway 流式逐 frame 测试（验证 chunk 是否逐个到达）",
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument(
        "api",
        nargs="?",
        default="openai",
        choices=["openai", "anthropic", "all"],
        help="测试接口（默认 openai）",
    )
    parser.add_argument("--host", default=os.environ.get("LLM_HOST", DEFAULT_HOST))
    parser.add_argument("--api-key", default=os.environ.get("LLM_API_KEY") or os.environ.get("API_KEY") or DEFAULT_API_KEY)
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--no-color", action="store_true")
    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()
    _set_color(not args.no_color)

    host = args.host.rstrip("/")
    model = args.model
    api_key = args.api_key

    if not api_key:
        print(Color.wrap("✗ 缺少 API Key", Color.RED), file=sys.stderr)
        return 2

    if args.api in ("openai", "all"):
        test_stream_openai(host, api_key, model)

    if args.api in ("anthropic", "all"):
        if args.api == "all":
            print(Color.wrap("=" * 70, Color.DIM))
            print()
        test_stream_anthropic(host, api_key, model)

    print(Color.wrap("=== 测试完成 ===", Color.CYAN))
    return 0


if __name__ == "__main__":
    sys.exit(main())
