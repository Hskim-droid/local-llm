#!/usr/bin/env python3
"""Drop a video, PPTX, and PDF → local Ollama writes a Word report."""
from __future__ import annotations

import argparse
import json
import os
import shlex
import subprocess
import sys
import time
from pathlib import Path

ROOT = Path(__file__).resolve().parent
if str(ROOT) not in sys.path:
    sys.path.insert(0, str(ROOT))

from extract import VIDEO, PPTX, extract_many
from hardware import inspect
from ollama_client import OllamaError, connect, load_config
from render import write_outputs
from structure import default_title, structure


def main(argv: list[str] | None = None) -> int:
    if hasattr(sys.stdout, "reconfigure"):
        try:
            sys.stdout.reconfigure(encoding="utf-8")
            sys.stderr.reconfigure(encoding="utf-8")
        except Exception:
            pass
    ap = argparse.ArgumentParser(
        prog="reportctl",
        description="영상·PPT·PDF를 로컬 모델로 한글 워드 보고서로. LG 그램 윈도우용.",
    )
    ap.add_argument("input", nargs="*", type=Path, help="mp4/m4a, pptx, pdf — 창에 끌어다 놓기")
    ap.add_argument("--out", type=Path, help="출력 폴더 (기본: 첫 파일 옆)")
    ap.add_argument("--model", help="Ollama 모델 태그 (프로필 기본값 대신)")
    ap.add_argument("--profile", choices=["gram16", "gram32", "mac24"], help="사양 프로필 강제")
    ap.add_argument("--status", action="store_true", help="RAM·GPU·모델 프로필만 출력하고 종료")
    ap.add_argument("--kind", choices=["auto", "slides", "minutes"], default="auto")
    ap.add_argument("--title", help="보고서 제목")
    ap.add_argument("--no-llm", action="store_true", help="모델 없이 추출·골격만")
    ap.add_argument("--pdf", action="store_true", help="PDF도 저장")
    ap.add_argument("--extras", action="store_true", help="md/html도 저장")
    ap.add_argument("--dump", action="store_true", help="추출 JSON만 저장")
    ap.add_argument("--no-open", action="store_true", help="끝나면 워드를 열지 않음")
    ap.add_argument("--no-pull", action="store_true", help="없는 모델을 받지 않음")
    args = ap.parse_args(argv)

    cfg = load_config(ROOT)
    machine = inspect(cfg, args.profile)
    if args.status:
        print(machine.summary())
        print(f"프로필={machine.profile_id}")
        print(f"받을 모델={', '.join(machine.profile.get('setup_models') or [machine.profile.get('pull') or ''])}")
        print(f"선호 순서={', '.join(machine.profile.get('model_pref') or [])}")
        print(f"컨텍스트={machine.profile.get('num_ctx')}  청크={machine.profile.get('chunk_chars')}자")
        note = machine.profile.get("notes_ko") or machine.profile.get("notes")
        if note:
            print(note)
        return 0

    dropped = list(args.input)
    if not dropped:
        dropped = prompt_drop()
        if not dropped:
            print("파일을 먼저 놓아 주세요. 예:  report  친 다음 영상·PPT·PDF를 끌어다 놓고 Enter", file=sys.stderr)
            return 2
    sources = [p.expanduser().resolve() for p in dropped]
    missing = [str(p) for p in sources if not p.exists()]
    if missing:
        print("파일 없음: " + ", ".join(missing), file=sys.stderr)
        return 2

    if len(sources) > 3:
        print("참고: 영상+PPT+PDF 세 개 정도가 적당합니다.", file=sys.stderr)

    t0 = time.monotonic()
    print(f"추출 중… {len(sources)}개 파일", flush=True)
    segs = extract_many(sources)
    if not segs:
        print("꺼낼 텍스트가 없습니다.", file=sys.stderr)
        return 1
    print(f"  세그먼트 {len(segs)}개", flush=True)

    out_dir = args.out or (sources[0].parent / f"{sources[0].stem}_report")
    out_dir = out_dir.expanduser().resolve()
    stem = "report"

    if args.dump:
        out_dir.mkdir(parents=True, exist_ok=True)
        dump = out_dir / "extract.json"
        dump.write_text(
            json.dumps([s.__dict__ for s in segs], ensure_ascii=False, indent=2),
            encoding="utf-8",
        )
        print(dump)
        return 0

    kind = args.kind
    if kind == "auto":
        exts = {p.suffix.lower() for p in sources}
        mixed = len(sources) > 1 or bool(exts & VIDEO)
        kind = "minutes" if mixed else ("slides" if exts & PPTX else "minutes")
    title = args.title or default_title(sources)

    print(machine.summary(), flush=True)
    client = None
    chunk_chars = int(machine.profile.get("chunk_chars") or 1000)
    if not args.no_llm:
        try:
            client = connect(cfg, args.model, pull=not args.no_pull, profile=machine.profile)
        except OllamaError as exc:
            print(str(exc), file=sys.stderr)
            return 1
        print(f"모델 {client.model}  컨텍스트 {client.options.get('num_ctx')}", flush=True)
    else:
        print("모델 없이 골격만 작성합니다.", flush=True)

    def prog(msg: str) -> None:
        print(f"  {msg}", flush=True)

    packed = structure(
        segs,
        title=title,
        kind=kind,
        client=client,
        chunk_chars=chunk_chars,
        progress=prog,
    )
    print("워드 작성 중…", flush=True)
    paths = write_outputs(packed, out_dir, stem, pdf=args.pdf, extras=args.extras)

    dt = time.monotonic() - t0
    docx = paths.get("docx")
    if docx:
        print(f"\n저장됨  {docx}", flush=True)
    for k, p in paths.items():
        if k != "docx":
            print(f"  {k}: {p}")
    print(f"완료 {dt:.1f}초")
    if docx and not args.no_open:
        spit_open(docx)
    return 0


def parse_drop_line(line: str) -> list[Path]:
    """터미널 드래그앤드롭이 붙인 경로(공백 이스케이프)를 파일 목록으로."""
    try:
        parts = shlex.split(line.strip())
    except ValueError:
        parts = line.strip().split()
    return [Path(p).expanduser() for p in parts if p]


def prompt_drop() -> list[Path]:
    if not sys.stdin.isatty():
        return []
    print("영상 · PPT · PDF 를 이 창에 끌어다 놓고 Enter", flush=True)
    try:
        line = sys.stdin.readline()
    except KeyboardInterrupt:
        print()
        return []
    return parse_drop_line(line)


def spit_open(path: Path) -> None:
    path = Path(path)
    try:
        if sys.platform == "darwin":
            subprocess.run(["open", str(path)], check=False)
        elif sys.platform.startswith("win"):
            os.startfile(str(path))  # type: ignore[attr-defined]
        else:
            subprocess.run(["xdg-open", str(path)], check=False)
    except OSError as exc:
        print(f"파일을 열지 못했습니다: {exc}", file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())
