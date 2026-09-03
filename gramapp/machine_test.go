package main

import "testing"

func TestMatchMachineProfileByRAM(t *testing.T) {
	m := decodeMachine(embeddedMachine)
	if len(m.Profiles) < 3 {
		t.Fatalf("profiles %d", len(m.Profiles))
	}
	g16 := matchMachineProfile(16, "windows", m.Profiles)
	if g16.ID != "gram16" || g16.Predict != 1200 || g16.Ctx != 4096 {
		t.Fatalf("16GB win %#v", g16)
	}
	g32 := matchMachineProfile(32, "windows", m.Profiles)
	if g32.ID != "gram32" || g32.Predict != 1100 {
		t.Fatalf("32GB win %#v", g32)
	}
	mac := matchMachineProfile(24, "darwin", m.Profiles)
	if mac.ID != "mac24" || mac.Pull[0] != "qwen3-14b" {
		t.Fatalf("mac24 %#v", mac)
	}
	bigMac := matchMachineProfile(32, "darwin", m.Profiles)
	if bigMac.ID != "gram32" {
		t.Fatalf("32GB mac should not be mac24: %#v", bigMac)
	}
}

func TestLlamaReleaseFromMachine(t *testing.T) {
	loadedMachine = decodeMachine(embeddedMachine)
	if llamaRelease() != "b10670" {
		t.Fatal(llamaRelease())
	}
}

func TestAvailMinSkips14B(t *testing.T) {
	loadedMachine = decodeMachine(embeddedMachine)
	m := loadedMachine
	h := hostSnap{Total: 24, Avail: 5, OS: "darwin"}
	got := matchHostProfile(h, m.Profiles)
	if got.ID == "mac24" {
		t.Fatalf("5GB free should skip mac24: %#v", got)
	}
	p := pickProfileHost(h)
	if len(p.Pull) != 1 || p.Pull[0] != "qwen3-8b" {
		t.Fatalf("derate to 8B: %#v", p)
	}
	if p.Ctx > 4096 || p.Predict > 800 {
		t.Fatalf("caps %#v", p)
	}
}

func TestDerateTightestFirst(t *testing.T) {
	m := decodeMachine(embeddedMachine)
	p := profile{ID: "mac24", Pull: []string{"qwen3-14b"}, Ctx: 8192, Chunk: 1600, Predict: 1400, ThreadsCap: 8}
	got := applyDerate(p, hostSnap{Avail: 3.2}, m.Derate)
	if got.Predict != 512 || got.Ctx != 2048 || got.Pull[0] != "qwen3-8b" {
		t.Fatalf("%#v", got)
	}
}

func TestPlentyRAMKeeps14B(t *testing.T) {
	loadedMachine = decodeMachine(embeddedMachine)
	p := pickProfileHost(hostSnap{Total: 24, Avail: 16, OS: "darwin"})
	if p.ID != "mac24" || p.Predict != 1400 {
		t.Fatalf("%#v", p)
	}
}
