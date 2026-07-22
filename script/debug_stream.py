import urllib.request
import json
import urllib.error

url = "http://localhost:3001/anthropic/v1/messages"
body = json.dumps({
    "model": "basic",
    "max_tokens": 1024,
    "messages": [{"role": "user", "content": "从1数到3"}],
    "stream": True
}).encode()

req = urllib.request.Request(url, data=body, headers={
    "Content-Type": "application/json",
    "Authorization": "Bearer sk-c7d79944f064315b0ce5a04c3c1daef632b43401765eea13f5fd391f0c6732c3",
    "anthropic-version": "2023-06-01"
})

try:
    resp = urllib.request.urlopen(req, timeout=30)
    count = 0
    for raw in resp:
        line = raw.decode("utf-8", errors="replace").rstrip("\r\n")
        print(repr(line), flush=True)
        count += 1
    resp.close()
    print(f"--- Total lines: {count} ---", flush=True)
except urllib.error.HTTPError as e:
    print(f"HTTP {e.code}: {e.read().decode()[:500]}", flush=True)
except Exception as e:
    print(f"Error: {e}", flush=True)
