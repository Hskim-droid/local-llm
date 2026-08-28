package main

// 예전 Ollama HTTP 호출은 llama.cpp로 옮겼습니다.
// 호출부 호환용 이름만 남깁니다.

func ollamaChatJSON(model, system, user string, ctx, predict int) (map[string]any, error) {
	return llamaChatJSON(model, system, user, ctx, predict)
}

func ollamaStop(_ string) { llamaStop() }
