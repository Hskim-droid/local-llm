package main

import (
	"fmt"
	"os/exec"
	"time"
)

func say(s string) { fmt.Println(s) }

func setup(p profile) error {
	first := !chatModelReady(p)
	say("")
	say("════════════════════════════════════════")
	say("  local-llm  " + appVersion)
	say("  " + T("banner.local"))
	say("  " + T("banner.report", reportIssuesURL))
	say("════════════════════════════════════════")
	if first {
		say("")
		say(T("first.title"))
		say(T("first.model"))
		say(T("first.wifi"))
		say(T("first.orig"))
		say(T("first.types"))
		say(T("first.old"))
	}
	say("")

	if first {
		h := snapshotHost()
		say(T("step.hw"))
		say(T("step.hw.ram2", h.Total, h.Avail, h.Load, gpuName()))
		say(T("step.hw.profile", profileLabel(p)))
		say(T("step.hw.tune", p.Ctx, p.Chunk, p.Predict))
		say(T("step.hw.where"))
		if p.ID == "gram16" {
			say(T("step.hw.16"))
		}
		if p.ID == "gram32" {
			say(T("step.hw.32"))
		}
		if av := availGB(); av < 5 {
			say(T("step.hw.low", av))
			say(T("step.hw.low2"))
			waitEnter(T("prompt.enter"))
		}
		say("")
		say(T("step.engine"))
	}
	if err := ensureVCRedist(); err != nil {
		return err
	}
	if err := ensureLlamaBin(); err != nil {
		return err
	}
	if first {
		say(T("step.ready"))
		say("")
		say(T("step.models"))
		say(T("step.models.hint"))
	}
	for i, id := range p.Pull {
		g, ok := specByID(id)
		if !ok {
			continue
		}
		if ggufReady(g) {
			if first {
				say(T("step.have", i+1, len(p.Pull), modelLabel(g)))
			}
			continue
		}
		if !first {
			say(T("step.get2", modelLabel(g)))
		} else {
			say(T("step.get", i+1, len(p.Pull), modelLabel(g)))
		}
		t0 := time.Now()
		if err := ensureGGUF(g); err != nil {
			if i == 0 {
				return fmt.Errorf("%s", T("err.model.dl"))
			}
			say(T("step.skip"))
			continue
		}
		say(T("step.done", time.Since(t0).Round(time.Second)))
	}
	if first {
		say("")
		say(T("step.prepared"))
		say("")
	}
	return nil
}

func specByID(id string) (ggufSpec, bool) {
	switch id {
	case gguf8b.ID:
		return gguf8b, true
	case gguf14b.ID:
		return gguf14b, true
	default:
		return ggufSpec{}, false
	}
}

func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}

var skipPause bool

func waitEnter(prompt string) {
	if skipPause {
		return
	}
	_ = promptLine(prompt)
}
