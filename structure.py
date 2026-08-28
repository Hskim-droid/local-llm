"""세그먼트 → 회의록 content.json. 모델은 빈칸만 채움."""
from __future__ import annotations

import re
from pathlib import Path

from extract import Segment, collect_numbers, hangul_ratio
from ollama_client import Ollama, qwen_user

SYS_CHUNK = (
    "너는 로컬 문서 정리기다. 원문에 있는 사실만 한국어 개조식으로 옮긴다. "
    "창작 금지. 없는 숫자·이름·일정을 만들지 말 것. 숫자는 원문 표기 그대로. "
    "명사형 종결(~함/~됨/~예정/~필요). 구어·완결형 문장 금지. "
    "일본어·영어 원문은 한국어로 직역·정리한다. 중간 영어 초안은 쓰지 말 것. "
    "JSON만 반환."
)

SYS_PACK = (
    "너는 로컬 문서 편집기다. 주어진 사실만 사용해 보고서를 주제별로 묶는다. "
    "슬라이드/페이지 하나당 섹션 하나를 만들지 말 것. 중복 문장은 합친다. "
    "사실에 없는 내용을 보태지 말 것. 숫자는 그대로. 명사형 종결. JSON만 반환."
)


def chunk_segments(segments: list[Segment], limit: int) -> list[list[Segment]]:
    """슬라이드·절은 합치지 않는다. 긴 페이지만 자른다."""
    batches: list[list[Segment]] = []
    for seg in segments:
        if len(seg.text) <= limit:
            batches.append([seg])
        else:
            for piece in _hard_split(seg, limit):
                batches.append([piece])
    return batches


def _hard_split(seg: Segment, limit: int) -> list[Segment]:
    text = seg.text
    parts: list[Segment] = []
    i = 0
    k = 1
    while i < len(text):
        parts.append(Segment(text[i : i + limit], seg.source, f"{seg.location}-{k}"))
        i += limit
        k += 1
    return parts


def structure(
    segments: list[Segment],
    *,
    title: str,
    kind: str,
    client: Ollama | None,
    chunk_chars: int,
    progress=None,
) -> dict:
    src = "\n".join(s.text for s in segments)
    already_ko = hangul_ratio(src) >= 0.45
    facts: list[dict] = []
    batches = chunk_segments(segments, chunk_chars)
    for i, batch in enumerate(batches, 1):
        if progress:
            progress(f"chunk {i}/{len(batches)}")
        blob = _batch_text(batch)
        if client is None:
            facts.append(_heuristic_chunk(blob, batch[0].location, translate=not already_ko))
        else:
            facts.append(_llm_chunk(client, blob, batch[0].location, already_ko))
    packed = _pack(facts, title, kind, client, src)
    missing = _missing_numbers(src, packed)
    if missing:
        packed.setdefault("appendix", {
            "band": "< 부록 · 사실 확인표 >",
            "note": "원문에 있던 숫자 중 본문에 안 실린 항목. 창작이 아니라 누락 표시.",
            "table": {
                "headers": ["#", "항목", "원문 숫자", "근거", "판정"],
                "widths": [6, 22, 28, 24, 20],
                "rows": [
                    [str(i), "수치", n, "원문", "확인 필요"]
                    for i, n in enumerate(missing[:12], 1)
                ],
            },
        })
    packed["closing"] = "끝."
    return packed


def _batch_text(batch: list[Segment]) -> str:
    return "\n\n".join(f"[{s.location}]\n{s.text}" for s in batch)


def _llm_chunk(client: Ollama, blob: str, loc: str, already_ko: bool) -> dict:
    user = (
        f"위치: {loc}\n이미한국어: {already_ko}\n"
        "다음 원문을 사실 목록으로 정리하라.\n"
        '{"location":"","heading_hint":"","facts":["명사형 문장"],'
        '"rows":[["구분","내용"]]}\n'
        "표가 없으면 rows는 []. heading_hint는 짧은 주제명.\n\n"
        f"{blob}"
    )
    try:
        obj = client.chat_json(SYS_CHUNK, qwen_user(user, client.model))
    except Exception:
        return _heuristic_chunk(blob, loc, translate=False)
    facts = [str(x).strip() for x in obj.get("facts") or [] if str(x).strip()]
    rows = []
    for r in obj.get("rows") or []:
        if isinstance(r, (list, tuple)) and len(r) >= 2:
            rows.append([str(r[0]).strip(), str(r[1]).strip()])
    if not facts:
        facts = _lines(blob)[:8]
    return {
        "location": str(obj.get("location") or loc),
        "heading_hint": str(obj.get("heading_hint") or "").strip(),
        "facts": facts[:12],
        "rows": rows[:8],
    }


def _heuristic_chunk(blob: str, loc: str, translate: bool) -> dict:
    lines = _lines(blob)
    facts = []
    for ln in lines[:12]:
        if translate and hangul_ratio(ln) < 0.3:
            facts.append(f"(원문) {ln}")
        else:
            facts.append(_nominal(ln))
    hint = ""
    for ln in blob.splitlines():
        s = ln.strip()
        if s and not s.startswith("[") and not _is_meta_line(s):
            hint = re.sub(r"^[\d\.]+\s*", "", s)
            hint = hint[:24]
            break
    return {"location": loc, "heading_hint": hint, "facts": facts, "rows": []}


def _pack(facts: list[dict], title: str, kind: str, client: Ollama | None, src: str = "") -> dict:
    compact = []
    for f in facts:
        compact.append({
            "location": f.get("location"),
            "heading_hint": f.get("heading_hint"),
            "facts": f.get("facts") or [],
            "rows": f.get("rows") or [],
        })
    if client is not None:
        user = (
            f"제목후보: {title}\n종류: {kind}\n"
            "사실만 사용해 주제별 회의록 JSON을 채워라. 모르면 '(원문에 없음)'. "
            "섹션은 최대 5개. 제목 예: 1. 개요  2. 핵심 내용  3. 수치·일정  4. 질의 및 논의.\n"
            '{"title":"","date":"","time":"","place":"","attendees":"","agenda":"",'
            '"exec":["명사형 3~5개"],'
            '"sections":[{"heading":"1. 개요","facts":["..."],"rows":[["구분","내용"]]}],'
            '"actions":["..."]}\n\n'
            f"{compact}"
        )
        try:
            meta = client.chat_json(SYS_PACK, qwen_user(user, client.model))
            return _from_meta(meta, title, kind, facts, src=src or _facts_src(facts))
        except Exception:
            pass
    return _from_meta({}, title, kind, facts, src=src or _facts_src(facts))


def _facts_src(facts: list[dict]) -> str:
    lines = []
    for f in facts:
        lines.extend(f.get("facts") or [])
    return "\n".join(lines)


def _from_meta(meta: dict, title: str, kind: str, facts: list[dict], src: str = "") -> dict:
    guessed = _guess_fields(src)
    t = str(meta.get("title") or title).strip() or title
    def pick(key: str, fallback: str) -> str:
        v = str(meta.get(key) or "").strip()
        if v and v != "(원문에 없음)":
            return v
        return guessed.get(key) or fallback

    date = pick("date", "(원문에 없음)")
    time = pick("time", "(원문에 없음)")
    place = pick("place", "(원문에 없음)")
    attendees = pick("attendees", "(원문에 없음)")
    agenda = pick("agenda", _guess_agenda(facts))
    exec_s = [str(x).strip() for x in (meta.get("exec") or []) if str(x).strip()]
    if not exec_s:
        exec_s = _exec_from_facts(facts)
    actions = [str(x).strip() for x in (meta.get("actions") or []) if str(x).strip()]
    sections = [
        {"heading": "0. Executive Summary", "blocks": [{"bullets": exec_s[:6]}]},
    ]
    grouped = _meta_sections(meta.get("sections") or [])
    if grouped:
        for i, g in enumerate(grouped[:5], 1):
            heading = str(g.get("heading") or f"{i}. 항목").strip()
            blocks = []
            facts_g = [str(x).strip() for x in (g.get("facts") or []) if str(x).strip()]
            if facts_g:
                blocks.append({"bullets": facts_g[:10]})
            rows_g = []
            for r in g.get("rows") or []:
                if isinstance(r, (list, tuple)) and len(r) >= 2:
                    rows_g.append([str(r[0]).strip(), str(r[1]).strip()])
            if rows_g:
                blocks.append({
                    "table": {
                        "headers": ["구분", "내용"],
                        "widths": [22, 78],
                        "firstcol_bold": True,
                        "rows": rows_g[:8],
                    }
                })
            if blocks:
                sections.append({"heading": heading, "blocks": blocks})
    else:
        for i, f in enumerate(facts, 1):
            hint = f.get("heading_hint") or f.get("location") or f"항목 {i}"
            heading = f"{i}. {hint}"
            blocks = []
            if f.get("facts"):
                blocks.append({"bullets": f["facts"]})
            if f.get("rows"):
                blocks.append({
                    "table": {
                        "headers": ["구분", "내용"],
                        "widths": [22, 78],
                        "firstcol_bold": True,
                        "rows": f["rows"],
                    }
                })
            if blocks:
                sections.append({"heading": heading, "blocks": blocks})
    if not grouped:
        discuss = _discussion(facts)
        if discuss:
            sections.append({"heading": f"{len(sections)}. 주요 질의 및 논의", "blocks": [{"bullets": discuss}]})
    if not actions:
        actions = ["원문 기준 후속 일정은 추가 확인이 필요함"]
    sections.append({
        "heading": f"{len(sections)}. 향후 계획",
        "blocks": [{"ordered": actions[:8]}],
    })
    label = "장표 보고서" if kind == "slides" else "회의록"
    if "회의록" not in t and "보고서" not in t:
        t = f"{t} {label}".strip()
    return {
        "title": t,
        "header": [
            ["일 시", date, "회의시간", time],
            ["장 소", place],
            ["참석자", attendees],
            ["주요 아젠다", agenda],
        ],
        "note": None,
        "sections": sections,
        "closing": "끝.",
    }


def _meta_sections(raw) -> list[dict]:
    out = []
    if not isinstance(raw, list):
        return out
    for g in raw:
        if isinstance(g, dict) and (g.get("facts") or g.get("rows") or g.get("heading")):
            out.append(g)
    return out


def _exec_from_facts(facts: list[dict]) -> list[str]:
    out = []
    for f in facts:
        for x in f.get("facts") or []:
            if _is_meta_line(x) or x.startswith(("일시", "장소", "참석", "日時", "場所")):
                continue
            out.append(x)
            if len(out) >= 5:
                return out
    return out or ["원문에서 핵심 결론을 특정하지 못함"]


def _discussion(facts: list[dict]) -> list[str]:
    keys = ("질의", "질문", "논의", "Q&A", "Q.", "問")
    out = []
    for f in facts:
        for x in f.get("facts") or []:
            if any(k in x for k in keys):
                out.append(x)
    return out[:8]


def _guess_agenda(facts: list[dict]) -> str:
    hints = [f.get("heading_hint") for f in facts if f.get("heading_hint")]
    if hints:
        return " / ".join(hints[:4])
    return "(원문에 없음)"


def _lines(blob: str) -> list[str]:
    out = []
    for ln in blob.splitlines():
        s = re.sub(r"^[\-\*\d\.\)\s]+", "", ln).strip()
        if len(s) >= 4 and not s.startswith("["):
            out.append(s)
    return out


def _guess_fields(src: str) -> dict:
    out = {"date": "", "time": "", "place": "", "attendees": "", "agenda": ""}
    tm = re.search(r"(\d{1,2}:\d{2})\s*[~\-–〜～]\s*(\d{1,2}:\d{2})", src)
    if tm:
        out["time"] = f"{tm.group(1)} ~ {tm.group(2)}"
    for ln in src.splitlines():
        s = ln.strip()
        if not s:
            continue
        if re.search(r"일시|日時|날짜", s) and not out["date"]:
            out["date"] = _after_label(s)
        elif re.search(r"장소|場所|장소", s) and not out["place"]:
            out["place"] = _after_label(s)
        elif re.search(r"참석|出席|참가", s) and not out["attendees"]:
            out["attendees"] = _after_label(s)
        elif re.search(r"아젠다|agenda|목적|議題", s, re.I) and not out["agenda"]:
            out["agenda"] = _after_label(s)
    return out


def _after_label(s: str) -> str:
    return re.sub(
        r"^(일시|日時|날짜|장소|場所|참석자?|出席|참가|아젠다|agenda|목적|議題)\s*[:：]?\s*",
        "",
        s,
        flags=re.I,
    ).strip()


def _is_meta_line(s: str) -> bool:
    return bool(re.search(r"^(일시|日時|장소|場所|참석|出席|아젠다)", s))


def _nominal(s: str) -> str:
    s = s.rstrip(" .。")
    if s.endswith(("함", "됨", "음", "임", "예정", "필요")):
        return s
    return s


def _missing_numbers(src: str, packed: dict) -> list[str]:
    dump = _flatten(packed)
    missing = []
    for n in collect_numbers(src):
        if _trivial(n):
            continue
        if n not in dump:
            missing.append(n)
    return missing


def _trivial(n: str) -> bool:
    core = re.sub(r"[^\d.]", "", n)
    if core in {"1", "2", "3", "4", "5"}:
        return True
    return False


def _flatten(obj) -> str:
    if isinstance(obj, str):
        return obj
    if isinstance(obj, dict):
        return " ".join(_flatten(v) for v in obj.values())
    if isinstance(obj, (list, tuple)):
        return " ".join(_flatten(v) for v in obj)
    return str(obj)


def default_title(paths: list[Path]) -> str:
    name = paths[0].stem if paths else "보고서"
    name = re.sub(r"[_]+", " ", name).strip()
    return name
