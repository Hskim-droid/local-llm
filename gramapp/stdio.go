package main

import "os"

func osStdout() *os.File { return os.Stdout }
func osStderr() *os.File { return os.Stderr }
