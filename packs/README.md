# 양식 팩

로컬LLM보고서는 **실행 파일 하나**에 용도만 갈아끼웁니다.

```
추출(코드) → 팩의 프롬프트로 JSON 채움 → 워드 렌더
```

정본은 `gramapp/packs/` 입니다. exe를 만들 때 안으로 들어갑니다. 푼 ZIP 안 `packs/`가 exe 옆에 있으면 그게 우선입니다.

| 폴더 | 용도 | 결과 |
|---|---|---|
| `보고서` | 장표·PDF 보고 | `*_보고서/보고서.docx` |
| `회의록` | 전사·녹취 | `*_회의록/회의록.docx` (whisper.cpp는 다음. 지금은 옆 `.txt`) |
| `번역` | 일한 직번역 | `*_번역/한국어번역.docx` (munseo-translator에서 이식) |

새 팩: `pack.json` + `chunk.txt` + `assemble.txt` 세 파일이면 됩니다.

```json
{
  "id": "보고서",
  "label": "보고서",
  "blurb": "장표·PDF·자료를 한글 보고서로",
  "title_suffix": "보고서",
  "out_suffix": "_보고서",
  "out_name": "보고서.docx",
  "body_heading": "내용"
}
```

