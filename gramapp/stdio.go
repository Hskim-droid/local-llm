package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func osStdout() *os.File { return os.Stdout }
func osStderr() *os.File { return os.Stderr }

var stdin = bufio.NewReader(os.Stdin)

func promptLine(prompt string) string {
	fmt.Print(prompt)
	s, _ := stdin.ReadString('\n')
	return strings.TrimSpace(s)
}
