package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---- currentStyleName ----

func TestCurrentStyleName_Valid(t *testing.T) {
	if len(chromaStyleNames) == 0 {
		t.Skip("chroma registry empty")
	}
	s := &appState{ChromaStyleIdx: 0}
	if got := currentStyleName(s); got != chromaStyleNames[0] {
		t.Fatalf("got %q, want %q", got, chromaStyleNames[0])
	}
}

func TestCurrentStyleName_NegativeIdx(t *testing.T) {
	s := &appState{ChromaStyleIdx: -1}
	if got := currentStyleName(s); got != "" {
		t.Fatalf("negative idx: got %q, want empty", got)
	}
}

func TestCurrentStyleName_OutOfRange(t *testing.T) {
	s := &appState{ChromaStyleIdx: len(chromaStyleNames)}
	if got := currentStyleName(s); got != "" {
		t.Fatalf("out-of-range idx: got %q, want empty", got)
	}
}

func TestCurrentStyleName_EmptyRegistry(t *testing.T) {
	saved := chromaStyleNames
	chromaStyleNames = nil
	t.Cleanup(func() { chromaStyleNames = saved })
	s := &appState{ChromaStyleIdx: 0}
	if got := currentStyleName(s); got != "" {
		t.Fatalf("empty registry: got %q, want empty", got)
	}
}

// ---- loadConfig ----

// redirectConfigDir points os.UserConfigDir at a temp dir for this
// test and returns the npad config file path inside it. Works on
// both darwin (HOME-based) and linux (XDG-based).
func redirectConfigDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "xdg"))
	p := configPath()
	if p == "" {
		t.Fatal("configPath returned empty after redirect")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func writeConfig(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadConfig_AtLimit(t *testing.T) {
	if len(chromaStyleNames) == 0 {
		t.Skip("chroma registry empty")
	}
	p := redirectConfigDir(t)
	cfg := npadConfig{ChromaStyleIdx: 0, RecentFiles: []string{"/a"}}
	body, _ := json.Marshal(cfg)
	// Place body at the end of a maxConfigBytes buffer; leading
	// bytes default to NUL — valid in a Reader bound check, and
	// we pre-pad with spaces so JSON parses cleanly.
	full := make([]byte, maxConfigBytes)
	for i := range full {
		full[i] = ' '
	}
	copy(full[maxConfigBytes-len(body):], body)
	writeConfig(t, p, full)

	s := &appState{ChromaStyleIdx: 5}
	loadConfig(s)
	if s.ChromaStyleIdx != 0 {
		t.Fatalf("ChromaStyleIdx not loaded: got %d", s.ChromaStyleIdx)
	}
	if len(s.RecentFiles) != 1 {
		t.Fatalf("RecentFiles: got %d, want 1", len(s.RecentFiles))
	}
}

func TestLoadConfig_OversizeFileRejected(t *testing.T) {
	p := redirectConfigDir(t)
	body := append([]byte(`{"chromaStyleIdx":1}`),
		make([]byte, maxConfigBytes+10)...)
	writeConfig(t, p, body)

	s := &appState{ChromaStyleIdx: 7, RecentFiles: []string{"keep"}}
	loadConfig(s)
	if s.ChromaStyleIdx != 7 {
		t.Fatalf("oversize file mutated state: ChromaStyleIdx=%d", s.ChromaStyleIdx)
	}
	if len(s.RecentFiles) != 1 || s.RecentFiles[0] != "keep" {
		t.Fatalf("oversize file mutated RecentFiles: %v", s.RecentFiles)
	}
}

func TestLoadConfig_BadJSON(t *testing.T) {
	p := redirectConfigDir(t)
	writeConfig(t, p, []byte("{not json"))

	s := &appState{ChromaStyleIdx: 3, RecentFiles: []string{"keep"}}
	loadConfig(s)
	if s.ChromaStyleIdx != 3 {
		t.Fatalf("bad JSON mutated ChromaStyleIdx: %d", s.ChromaStyleIdx)
	}
	if len(s.RecentFiles) != 1 || s.RecentFiles[0] != "keep" {
		t.Fatalf("bad JSON mutated RecentFiles: %v", s.RecentFiles)
	}
}

func TestLoadConfig_RecentPathTooLong(t *testing.T) {
	p := redirectConfigDir(t)
	long := strings.Repeat("a", maxPathBytes+1)
	cfg := npadConfig{
		ChromaStyleIdx: 0,
		RecentFiles:    []string{"/ok", long, "/also-ok"},
	}
	body, _ := json.Marshal(cfg)
	writeConfig(t, p, body)

	s := &appState{}
	loadConfig(s)
	if len(s.RecentFiles) != 2 {
		t.Fatalf("got %d recents, want 2 (long path dropped): %v",
			len(s.RecentFiles), s.RecentFiles)
	}
	if s.RecentFiles[0] != "/ok" || s.RecentFiles[1] != "/also-ok" {
		t.Fatalf("wrong recents kept: %v", s.RecentFiles)
	}
}

func TestLoadConfig_StyleIdxOutOfRangeIgnored(t *testing.T) {
	p := redirectConfigDir(t)
	cfg := npadConfig{ChromaStyleIdx: len(chromaStyleNames) + 100}
	body, _ := json.Marshal(cfg)
	writeConfig(t, p, body)

	s := &appState{ChromaStyleIdx: 4}
	loadConfig(s)
	if s.ChromaStyleIdx != 4 {
		t.Fatalf("out-of-range idx applied: got %d, want 4", s.ChromaStyleIdx)
	}
}

func TestLoadConfig_NilState(t *testing.T) {
	loadConfig(nil) // must not panic
}

func TestLoadConfig_MissingFile(t *testing.T) {
	redirectConfigDir(t) // dir exists, file does not
	s := &appState{ChromaStyleIdx: 9, RecentFiles: []string{"k"}}
	loadConfig(s)
	if s.ChromaStyleIdx != 9 || len(s.RecentFiles) != 1 {
		t.Fatalf("missing file mutated state: %+v", s)
	}
}
