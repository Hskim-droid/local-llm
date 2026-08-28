"""첫 실행: 사양 보고 필요한 것을 순서대로 받는다. 안내는 한국어."""
from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from hardware import inspect
from ollama_client import OllamaError, ensure_server, load_config, pull_model, _get

STAMP = ROOT / ".setup-ok.json"


def _utf8() -> None:
    if hasattr(sys.stdout, "reconfigure"):
        try:
            sys.stdout.reconfigure(encoding="utf-8")
            sys.stderr.reconfigure(encoding="utf-8")
        except Exception:
            pass


def say(msg: str) -> None:
    print(msg, flush=True)


def banner() -> None:
    say("")
    say("══════════════════════════════════════")
    say("  로컬 보고서  ·  LG 그램 / 윈도우")
    say("  영상 + PPT + PDF  →  한글 워드")
    say("══════════════════════════════════════")
    say("")


def pip_cmd() -> list[str]:
    return [sys.executable, "-m", "pip"]


def ollama_models(host: str) -> list[str]:
    try:
        tags = _get(f"{host.rstrip('/')}/api/tags", 5)
    except OllamaError:
        return []
    return [m.get("name", "") for m in tags.get("models", [])]


def model_present(name: str, installed: list[str]) -> bool:
    if name in installed:
        return True
    return any(n.startswith(name) for n in installed)


def wait_ollama_binary() -> str:
    while True:
        path = shutil.which("ollama")
        if path:
            return path
        say("  Ollama가 아직 PATH에 없습니다.")
        say("  1) 브라우저에서 엽니다:  https://ollama.com/download")
        say("  2) Windows 설치파일을 받아 설치합니다.")
        say("     끝나면 화면 오른쪽 아래 트레이에 라마 아이콘이 뜹니다.")
        say("  3) 이 창으로 돌아와 Enter 를 누르세요.")
        try:
            input("     설치 끝났으면 Enter > ")
        except EOFError:
            raise SystemExit("Ollama가 필요합니다.") from None


def install_python_packages() -> None:
    req = ROOT / "requirements.txt"
    if not req.exists():
        return
    say("  패키지를 확인합니다. 없으면 받습니다 (처음만, 몇 분).")
    r = subprocess.run(
        pip_cmd() + ["install", "-r", str(req), "-q"],
    )
    if r.returncode != 0:
        say("  pip 설치에 실패했습니다. 아래를 직접 실행해 보세요:")
        say(f"    {sys.executable} -m pip install -r \"{req}\"")
        raise SystemExit(1)


def setup(profile_id: str | None = None, force: bool = False) -> int:
    _utf8()
    banner()
    cfg = load_config(ROOT)
    machine = inspect(cfg, profile_id)
    p = machine.profile
    host = cfg.get("host", "http://127.0.0.1:11434")
    wanted = list(p.get("setup_models") or [p.get("pull") or "qwen3:8b"])

    say("[1/5] 이 노트북 사양")
    say(f"      RAM  {machine.ram_gb:.0f} GB")
    say(f"      GPU  {machine.gpu or '내장 그래픽 (VRAM 없음)'}")
    say(f"      프로필  {p.get('label', machine.profile_id)}")
    if machine.profile_id == "gram16":
        say("      → 16GB 그램입니다. 8B 모델만 씁니다. 14B는 올리지 않습니다.")
    elif machine.profile_id == "gram32":
        say("      → 32GB입니다. 8B를 받은 뒤, 여유 있으면 14B도 받습니다.")
    else:
        say("      → 맥 24GB 프로필입니다. 14B를 씁니다.")
    note = p.get("notes_ko") or p.get("notes")
    if note:
        say(f"      {note}")
    say("")

    if not force and STAMP.exists():
        try:
            prev = json.loads(STAMP.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            prev = {}
        have = []
        try:
            ensure_server(host)
            have = ollama_models(host)
        except OllamaError:
            have = []
        if prev.get("profile") == machine.profile_id and all(
            model_present(m, have) for m in wanted
        ):
            say("이전 설치가 그대로입니다. 준비 단계를 건너뜁니다.")
            say("")
            return 0

    say("[2/5] Python")
    say(f"      {sys.version.split()[0]}  ({sys.executable})")
    say("")

    say("[3/5] 파이썬 패키지")
    install_python_packages()
    say("      완료")
    say("")

    say("[4/5] Ollama (로컬 AI 엔진)")
    wait_ollama_binary()
    try:
        ensure_server(host)
    except OllamaError as exc:
        say(f"      서버를 못 켰습니다: {exc}")
        say("      Ollama 앱을 시작 메뉴에서 한 번 실행한 뒤 다시 돌려 주세요.")
        return 1
    say("      서버가 켜졌습니다. 문서는 이 노트북 밖으로 나가지 않습니다.")
    say("")

    say("[5/5] 모델 받기  (Wi-Fi, 전원 어댑터 연결을 권합니다)")
    have = ollama_models(host)
    total = len(wanted)
    for i, name in enumerate(wanted, 1):
        gb = {"qwen3:8b": "약 5GB", "qwen3:14b": "약 9GB", "exaone3.5:7.8b": "약 5GB"}.get(
            name, ""
        )
        extra = f" · {gb}" if gb else ""
        if model_present(name, have):
            say(f"      ({i}/{total}) {name}  이미 있습니다. 건너뜁니다.")
            continue
        say(f"      ({i}/{total}) {name}{extra}  받는 중… 진행률은 아래에 나옵니다.")
        t0 = time.monotonic()
        try:
            pull_model(name)
        except OllamaError as exc:
            say(f"      실패: {exc}")
            if i == 1:
                return 1
            say("      이 모델은 건너뛰고 이미 받은 모델로 진행합니다.")
            continue
        say(f"      완료 ({time.monotonic() - t0:.0f}초)")
        have = ollama_models(host)
    say("")

    STAMP.write_text(
        json.dumps(
            {
                "profile": machine.profile_id,
                "models": wanted,
                "ram_gb": round(machine.ram_gb, 1),
            },
            ensure_ascii=False,
            indent=2,
        ),
        encoding="utf-8",
    )
    say("준비됐습니다.")
    say("  파워셸에  report  를 치고 스페이스,")
    say("  탐색기에서 영상 · PPT · PDF 를 이 창으로 끌어다 놓고 Enter.")
    say("")
    return 0


def main(argv: list[str] | None = None) -> int:
    args = argv if argv is not None else sys.argv[1:]
    force = "--force" in args
    profile = None
    if "--profile" in args:
        i = args.index("--profile")
        if i + 1 < len(args):
            profile = args[i + 1]
    return setup(profile, force=force)


if __name__ == "__main__":
    raise SystemExit(main())
