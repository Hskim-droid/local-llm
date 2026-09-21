# 개인정보 · 로컬 처리

로컬LLM의 **Go 엔진**은 작업 파일을 이 노트북에서만 읽습니다. 아래 보장은 `gramapp/` 엔진에 해당합니다. 저장소의 `agent-harness/`는 별도의 실험용 Python 경로이며, fixture 의존성과 선택적 loopback Ollama-compatible 번역 경계를 따릅니다.

- 원본 PPT·PDF·영상·전사는 수정하지 않습니다. 결과는 원본 옆 새 폴더에만 씁니다.
- 문장 생성은 이 PC의 llama.cpp만 씁니다. 클라우드 번역·채팅 API를 쓰지 않습니다.
- 계정 로그인, 사용량 수집, 텔레메트리가 없습니다. 오류도 자동으로 안 올라갑니다.
- 오류 신고는 https://github.com/Hskim-droid/local-llm/issues 로 주세요. 원문은 보내지 말고, 프로그램이 만든 `오류.txt`만 주세요.
- Go 엔진이 인터넷을 쓰는 경우는 **도구를 받을 때**뿐입니다. llama.cpp, Qwen3 GGUF, 필요할 때만 그림 GGUF·whisper.cpp·ffmpeg를 받습니다. 작업 파일은 나가지 않습니다. 하네스의 선택적 adapter는 별도 설정이며, loopback 주소를 강제하지만 실행 중인 서버가 실제로 로컬인지 자체 검증하지 않습니다.
- Windows SmartScreen 경고는 코드 서명이 없어서 뜹니다. 마이크로소프트에 파일을 보내지 않습니다.

번역은 **번역** 팩으로 처리합니다. 기본은 한국어입니다.
