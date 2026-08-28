"""localhost Ollama만 호출. 서버가 꺼져 있으면 켠다."""
from __future__ import annotations

import json
import os
import shutil
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path


def load_config(root: Path | None = None) -> dict:
    path = (root or Path(__file__).resolve().parent) / "config.json"
    return json.loads(path.read_text(encoding="utf-8"))


class OllamaError(RuntimeError):
    pass


class Ollama:
    def __init__(self, host: str, model: str, options: dict, timeout: int, keep_alive: str):
        self.host = host.rstrip("/")
        self.model = model
        self.options = options
        self.timeout = timeout
        self.keep_alive = keep_alive

    def chat_json(self, system: str, user: str) -> dict:
        payload = {
            "model": self.model,
            "stream": False,
            "format": "json",
            "keep_alive": self.keep_alive,
            "options": self.options,
            "messages": [
                {"role": "system", "content": system},
                {"role": "user", "content": user},
            ],
        }
        data = _post(f"{self.host}/api/chat", payload, self.timeout)
        raw = (data.get("message") or {}).get("content") or ""
        return _parse_json(raw)


def connect(cfg: dict, model: str | None = None, pull: bool = True, profile: dict | None = None) -> Ollama:
    host = cfg["host"]
    p = profile or {}
    pref = list(p.get("model_pref") or cfg.get("model_pref") or ["qwen3:8b"])
    tags = ensure_server(host)
    names = [m.get("name", "") for m in tags.get("models", [])]
    chosen = _pick_model(model, pref, names)
    if not chosen:
        want = model or p.get("pull") or pref[0]
        if not pull:
            raise OllamaError(f"no model installed. run once: ollama pull {want}")
        print(f"no model yet. downloading {want} (one-time, several GB)…", flush=True)
        pull_model(want)
        tags = _get(f"{host}/api/tags", 8)
        names = [m.get("name", "") for m in tags.get("models", [])]
        chosen = _pick_model(model or want, pref, names)
    if not chosen:
        raise OllamaError("model not found. check `ollama list`.")
    options = {
        "temperature": cfg.get("temperature", 0.1),
        "num_ctx": int(p.get("num_ctx") or cfg.get("num_ctx") or 4096),
        "num_predict": int(p.get("num_predict") or cfg.get("num_predict") or 800),
        "num_gpu": -1,
        "num_thread": 0,
    }
    return Ollama(
        host,
        chosen,
        options,
        int(p.get("timeout_sec") or cfg.get("timeout_sec") or 180),
        cfg.get("keep_alive", "5m"),
    )


def ensure_server(host: str) -> dict:
    try:
        return _get(f"{host}/api/tags", 3)
    except OllamaError:
        pass
    binary = shutil.which("ollama")
    if not binary:
        raise OllamaError("Ollama is not installed. Get it from https://ollama.com")
    print("Ollama is not running. starting the server…", flush=True)
    kw: dict = {"stdout": subprocess.DEVNULL, "stderr": subprocess.DEVNULL}
    if os.name == "nt":
        kw["creationflags"] = (
            getattr(subprocess, "CREATE_NEW_PROCESS_GROUP", 0)
            | getattr(subprocess, "DETACHED_PROCESS", 0)
            | getattr(subprocess, "CREATE_NO_WINDOW", 0)
        )
    else:
        kw["start_new_session"] = True
    subprocess.Popen([binary, "serve"], **kw)
    last = None
    for _ in range(40):
        try:
            return _get(f"{host}/api/tags", 2)
        except OllamaError as exc:
            last = exc
            time.sleep(0.4)
    raise OllamaError(f"could not start Ollama ({host}). {last}")


def stop_model(name: str) -> None:
    binary = shutil.which("ollama")
    if not binary or not name:
        return
    subprocess.run([binary, "stop", name], capture_output=True, timeout=30)


def pull_model(name: str) -> None:
    binary = shutil.which("ollama")
    if not binary:
        raise OllamaError("Ollama is not installed.")
    try:
        subprocess.run([binary, "pull", name], check=True)
    except subprocess.CalledProcessError as exc:
        raise OllamaError(f"download failed: ollama pull {name}") from exc


def _pick_model(explicit: str | None, pref: list[str], names: list[str]) -> str | None:
    def has(name: str) -> bool:
        return name in names or any(n.split(":")[0] == name.split(":")[0] and name in n for n in names)

    if explicit:
        if explicit in names:
            return explicit
        for n in names:
            if n.startswith(explicit):
                return n
        raise OllamaError(f"model not installed: {explicit}  (have: {', '.join(names) or 'none'})")
    for p in pref:
        if p in names:
            return p
        for n in names:
            if n.startswith(p):
                return n
    for n in names:
        low = n.lower()
        if "qwen" in low and ("8b" in low or "7b" in low or "14b" in low):
            return n
        if "exaone" in low:
            return n
    return names[0] if names else None


def qwen_user(text: str, model: str) -> str:
    if "qwen3" in model.lower():
        return "/no_think\n" + text
    return text


def _parse_json(raw: str) -> dict:
    text = raw.strip()
    text = _strip_think(text)
    if text.startswith("```"):
        text = text.strip("`")
        if text.startswith("json"):
            text = text[4:]
        text = text.strip()
    start, end = text.find("{"), text.rfind("}")
    if start < 0 or end <= start:
        raise OllamaError("모델이 JSON을 반환하지 않았습니다.")
    try:
        obj = json.loads(text[start : end + 1])
    except json.JSONDecodeError as exc:
        raise OllamaError(f"JSON 파싱 실패: {exc}") from exc
    if not isinstance(obj, dict):
        raise OllamaError("JSON 객체가 아닙니다.")
    return obj


def _strip_think(text: str) -> str:
    if "</think>" in text:
        return text.split("</think>", 1)[1].strip()
    return text


def _get(url: str, timeout: int) -> dict:
    req = urllib.request.Request(url, method="GET")
    return _request(req, timeout)


def _post(url: str, payload: dict, timeout: int) -> dict:
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        url, data=body, method="POST", headers={"Content-Type": "application/json"}
    )
    return _request(req, timeout)


def _request(req: urllib.request.Request, timeout: int) -> dict:
    try:
        with urllib.request.urlopen(req, timeout=timeout) as res:
            raw = res.read().decode("utf-8", errors="replace")
    except urllib.error.URLError as exc:
        raise OllamaError(str(exc.reason if hasattr(exc, "reason") else exc)) from exc
    try:
        return json.loads(raw)
    except json.JSONDecodeError as exc:
        raise OllamaError("Ollama 응답이 JSON이 아닙니다.") from exc
