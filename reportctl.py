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
        description="Turn a video, PPTX, and PDF into a Word report with local Ollama 14B",
    )
    ap.add_argument("input", nargs="*", type=Path, help="mp4/m4a, pptx, pdf — drop onto the terminal")
    ap.add_argument("--out", type=Path, help="output folder (default: next to the first file)")
    ap.add_argument("--model", help="Ollama model tag (default: qwen3:14b if installed)")
    ap.add_argument("--kind", choices=["auto", "slides", "minutes"], default="auto")
    ap.add_argument("--title", help="report title")
    ap.add_argument("--no-llm", action="store_true", help="extract + skeleton only, skip the model")
    ap.add_argument("--pdf", action="store_true", help="also write PDF")
    ap.add_argument("--extras", action="store_true", help="also write md/html")
    ap.add_argument("--dump", action="store_true", help="write extract JSON only")
    ap.add_argument("--no-open", action="store_true", help="do not open the Word file when done")
    ap.add_argument("--no-pull", action="store_true", help="do not download a model if missing")
    args = ap.parse_args(argv)

    dropped = list(args.input)
    if not dropped:
        dropped = prompt_drop()
        if not dropped:
            print("Drop files first. Example:  report   then drag a video, PPTX, and PDF, then Enter", file=sys.stderr)
            return 2
    sources = [p.expanduser().resolve() for p in dropped]
    missing = [str(p) for p in sources if not p.exists()]
    if missing:
        print("file not found: " + ", ".join(missing), file=sys.stderr)
        return 2

    if len(sources) > 3:
        print("note: this script is meant for about 3 files (video + PPTX + PDF).", file=sys.stderr)

    t0 = time.monotonic()
    print(f"extracting {len(sources)} file(s)…", flush=True)
    segs = extract_many(sources)
    if not segs:
        print("no text extracted.", file=sys.stderr)
        return 1
    print(f"  {len(segs)} segment(s)", flush=True)

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

    client = None
    if not args.no_llm:
        cfg = load_config(ROOT)
        try:
            client = connect(cfg, args.model, pull=not args.no_pull)
        except OllamaError as exc:
            print(str(exc), file=sys.stderr)
            return 1
        print(f"model {client.model}  ctx={cfg.get('num_ctx')}", flush=True)
        chunk_chars = int(cfg.get("chunk_chars", 1600))
    else:
        cfg = load_config(ROOT)
        chunk_chars = int(cfg.get("chunk_chars", 1600))
        print("skeleton only (no model)", flush=True)

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
    print("writing Word…", flush=True)
    paths = write_outputs(packed, out_dir, stem, pdf=args.pdf, extras=args.extras)

    dt = time.monotonic() - t0
    docx = paths.get("docx")
    if docx:
        print(f"\nwrote  {docx}", flush=True)
    for k, p in paths.items():
        if k != "docx":
            print(f"  {k}: {p}")
    print(f"done {dt:.1f}s")
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
    print("Drop a video, PPTX, and PDF here, then press Enter", flush=True)
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
        print(f"could not open file: {exc}", file=sys.stderr)


if __name__ == "__main__":
    raise SystemExit(main())
