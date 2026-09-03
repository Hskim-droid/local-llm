package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	setUTF8Console()
	os.Setenv("PYTHONUTF8", "1")

	packArg := ""
	langArg := ""
	harvestOnly := false
	force8b := false
	var rawFiles []string
	for i := 1; i < len(os.Args); i++ {
		a := os.Args[i]
		switch {
		case a == "--harvest":
			harvestOnly = true
		case a == "--8b":
			force8b = true
		case a == "--yes":
			skipPause = true
		case a == "--lang" && i+1 < len(os.Args):
			langArg = os.Args[i+1]
			i++
		case strings.HasPrefix(a, "--lang="):
			langArg = strings.TrimPrefix(a, "--lang=")
		case a == "--pack" && i+1 < len(os.Args):
			packArg = os.Args[i+1]
			i++
		case strings.HasPrefix(a, "--pack="):
			packArg = strings.TrimPrefix(a, "--pack=")
		case strings.HasPrefix(a, "-"):
			continue
		default:
			rawFiles = append(rawFiles, a)
		}
	}
	setUILang(resolveLang(langArg))
	if langArg != "" {
		saveLang(langArg)
	}

	var files []string
	for _, f := range rawFiles {
		if abs, err := filepath.Abs(f); err == nil {
			if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
				files = append(files, abs)
			}
		}
	}

	loadMachineCatalog()
	pin8B = force8b
	p := liveProfile()
	if harvestOnly {
		if len(files) == 0 {
			say(T("harvest.need"))
			os.Exit(2)
		}
		out, err := runHarvestOnly(files)
		if err != nil {
			say(T("job.fail", err.Error()))
			reportFailure(p, "", files, err)
			os.Exit(1)
		}
		say(T("harvest.ok"))
		say("  " + out)
		return
	}

	if err := setup(p); err != nil {
		say(T("err.prefix", err.Error()))
		reportFailure(p, "", files, err)
		pause()
		os.Exit(1)
	}

	model, label, err := pickChatGGUF(p)
	if err != nil {
		say(T("err.prefix", err.Error()))
		reportFailure(p, "", files, err)
		pause()
		os.Exit(1)
	}
	say(T("using.model", label))
	h := snapshotHost()
	if h.Avail < 4 {
		say(T("mem.low"))
	}

	if packArg != "" {
		pk, err := pickPack(packArg)
		if err != nil {
			say(T("err.prefix", err.Error()))
			reportFailure(p, packArg, files, err)
			pause()
			os.Exit(1)
		}
		if err := runOneJob(files, p, model, pk, true); err != nil {
			os.Exit(1)
		}
		pause()
		return
	}

	runLoop(files, p, model)
}

func runLoop(pending []string, p profile, model string) {
	for {
		packs, err := catalogPacks()
		if err != nil {
			say(T("err.prefix", err.Error()))
			reportFailure(p, "", pending, err)
			pause()
			return
		}
		sayCanDo(packs)
		line := strings.TrimSpace(promptLine("> "))
		if isQuit(line) || (skipPause && line == "") {
			say(T("bye"))
			return
		}
		if line == "" {
			continue
		}
		pk, ok := matchPack(packs, line)
		if !ok {
			say(T("cando.only"))
			continue
		}
		pk, err = ensurePackLoaded(pk)
		if err != nil {
			say(T("err.prefix", err.Error()))
			reportFailure(p, pk.ID, pending, err)
			continue
		}
		files := pending
		pending = nil
		if err := runOneJob(files, p, model, pk, false); err != nil {
			continue
		}
	}
}

func runOneJob(files []string, p profile, model string, pk pack, failHard bool) error {
	say(T("job.pack", packLabel(pk)))
	if len(files) == 0 {
		files = pickFiles()
	}
	if len(files) == 0 {
		raw := strings.TrimSpace(promptLine(T("prompt.file")))
		if raw != "" {
			if abs, err := filepath.Abs(raw); err == nil {
				if st, err := os.Stat(abs); err == nil && st.Mode().IsRegular() {
					files = []string{abs}
				}
			}
		}
	}
	if len(files) == 0 {
		say(T("job.nofile"))
		if failHard {
			say(T("job.nofile.bat"))
			say(T("job.nofile.drop"))
			pause()
		}
		return fmt.Errorf("no files")
	}
	if len(files) > 1 {
		say(T("job.onebyone", len(files)))
	}
	p = liveProfile()
	if len(p.Pull) == 1 && p.Pull[0] == "qwen3-8b" && !ggufReady(gguf8b) {
		say(T("step.hw.derate"))
		_ = ensureGGUF(gguf8b)
	}
	if m, lab, err := pickChatGGUF(p); err == nil {
		model = m
		say(T("using.model", lab))
		h := snapshotHost()
		say(T("step.hw.ram2", h.Total, h.Avail, h.Load, gpuName()))
		say(T("step.hw.tune", p.Ctx, p.Chunk, p.Predict))
	}
	var firstErr error
	for i, f := range files {
		if len(files) > 1 {
			say(T("job.file", i+1, len(files), filepath.Base(f)))
		}
		t0 := time.Now()
		out, err := runFiles([]string{f}, p, model, pk)
		llamaStop()
		if err != nil {
			say(T("job.fail", err.Error()))
			say(T("job.orig"))
			reportFailure(p, pk.ID, []string{f}, err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		say("")
		say(T("job.done"))
		say("  " + out)
		say(T("job.time", time.Since(t0).Round(time.Second)))
		reveal(out)
	}
	if failHard && firstErr != nil {
		pause()
	}
	return firstErr
}

func pause() {
	if skipPause {
		return
	}
	_ = promptLine(T("prompt.quit"))
}

func mkdirAll(p string) error {
	return os.MkdirAll(p, 0755)
}
