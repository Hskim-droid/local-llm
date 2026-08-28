package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const ollamaHost = "http://127.0.0.1:11434"

type chatReq struct {
	Model     string    `json:"model"`
	Stream    bool      `json:"stream"`
	Format    string    `json:"format"`
	KeepAlive string    `json:"keep_alive"`
	Options   chatOpt   `json:"options"`
	Messages  []chatMsg `json:"messages"`
}

type chatOpt struct {
	Temperature float64 `json:"temperature"`
	NumCtx      int     `json:"num_ctx"`
	NumPredict  int     `json:"num_predict"`
}

type chatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResp struct {
	Message chatMsg `json:"message"`
}

func ollamaTags() ([]string, error) {
	resp, err := http.Get(ollamaHost + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var wrap struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrap); err != nil {
		return nil, err
	}
	var names []string
	for _, m := range wrap.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

func hasModel(names []string, want string) bool {
	for _, n := range names {
		if n == want || strings.HasPrefix(n, want) {
			return true
		}
	}
	return false
}

func pickInstalled(pref []string, names []string) string {
	for _, p := range pref {
		if hasModel(names, p) {
			for _, n := range names {
				if n == p || strings.HasPrefix(n, p) {
					return n
				}
			}
		}
	}
	if len(names) > 0 {
		return names[0]
	}
	return ""
}

func ollamaChatJSON(model, system, user string, ctx, predict int) (map[string]any, error) {
	if strings.Contains(strings.ToLower(model), "qwen3") && !strings.Contains(user, "/no_think") {
		user = "/no_think\n" + user
	}
	body, _ := json.Marshal(chatReq{
		Model:     model,
		Stream:    false,
		Format:    "json",
		KeepAlive: "2m",
		Options:   chatOpt{Temperature: 0.1, NumCtx: ctx, NumPredict: predict},
		Messages: []chatMsg{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	})
	cli := &http.Client{Timeout: 240 * time.Second}
	resp, err := cli.Post(ollamaHost+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var cr chatResp
	if err := json.Unmarshal(raw, &cr); err != nil {
		return nil, fmt.Errorf("ollama 응답 오류")
	}
	return parseJSONObj(cr.Message.Content)
}

func parseJSONObj(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "</think>"); i >= 0 {
		s = strings.TrimSpace(s[i+8:])
	}
	a, b := strings.Index(s, "{"), strings.LastIndex(s, "}")
	if a < 0 || b <= a {
		return nil, fmt.Errorf("JSON 없음")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(s[a:b+1]), &m); err != nil {
		return nil, err
	}
	return m, nil
}

func ollamaPull(name string) error {
	cmd := exec.Command("ollama", "pull", name)
	cmd.Stdout = osStdout()
	cmd.Stderr = osStderr()
	return cmd.Run()
}

func ollamaStop(name string) {
	_ = exec.Command("ollama", "stop", name).Run()
}

func ollamaServe() {
	cmd := exec.Command("ollama", "serve")
	_ = cmd.Start()
}

func waitOllama(sec int) bool {
	for i := 0; i < sec*2; i++ {
		if _, err := ollamaTags(); err == nil {
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}
