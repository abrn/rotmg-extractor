package download

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"rotmg-extractor/internal/logx"
)

func md5hex(b []byte) string {
	s := md5.Sum(b)
	return hex.EncodeToString(s[:])
}

func gzipped(b []byte) []byte {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

// newCDN serves a manifest plus gzip-only assets (plain paths 404), mimicking
// the real CDN. files maps manifest path -> content.
func newCDN(t *testing.T, files map[string][]byte) *httptest.Server {
	t.Helper()
	var m Manifest
	for p, c := range files {
		m.Files = append(m.Files, File{Path: p, Checksum: md5hex(c), Size: int64(len(c))})
	}
	manifestJSON, _ := json.Marshal(m)

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksum.json") {
			w.Write(manifestJSON)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".gz") {
			// r.URL.Path is already unescaped by net/http.
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/build/"), ".gz")
			if c, ok := files[key]; ok {
				w.Write(gzipped(c))
				return
			}
		}
		http.Error(w, "not found", http.StatusForbidden) // plain paths unavailable
	}))
}

func TestClientFiles(t *testing.T) {
	files := map[string][]byte{
		"baselib.dll":                         []byte("hello baselib"),
		"RotMG Exalt_Data/app.info":           []byte("DECA\nRotMGExalt"), // space + subdir
		"RotMG Exalt_Data/globalgamemanagers": bytes.Repeat([]byte("X"), 5000),
	}
	srv := newCDN(t, files)
	defer srv.Close()

	dest := t.TempDir()
	log := logx.New(logx.LevelError, false)
	if _, err := ClientFiles(context.Background(), log, srv.URL+"/build", dest, Options{}); err != nil {
		t.Fatalf("ClientFiles: %v", err)
	}
	for p, want := range files {
		got, err := os.ReadFile(filepath.Join(dest, filepath.FromSlash(p)))
		if err != nil {
			t.Errorf("missing %s: %v", p, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: content mismatch", p)
		}
	}
}

func TestChecksumMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksum.json") {
			json.NewEncoder(w).Encode(Manifest{Files: []File{{Path: "a.bin", Checksum: "deadbeef"}}})
			return
		}
		if strings.HasSuffix(r.URL.Path, ".gz") {
			w.Write(gzipped([]byte("actual content")))
			return
		}
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := ClientFiles(context.Background(), logx.New(logx.LevelError, false), srv.URL+"/build", t.TempDir(), Options{})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch error, got %v", err)
	}
}

func TestClientFilesFilterAndIncremental(t *testing.T) {
	files := map[string][]byte{
		"GameAssembly.dll":                    []byte("game assembly"),
		"RotMG Exalt_Data/resources.assets":   []byte("asset data"),
		"RotMG Exalt_Data/resources.resource": bytes.Repeat([]byte("Z"), 10000), // big, non-essential
		"baselib.dll":                         []byte("baselib"),                // non-essential
	}
	srv := newCDN(t, files)
	defer srv.Close()

	// Filter: only GameAssembly.dll + *.assets (essential-ish).
	filter := func(p string) bool {
		b := filepath.Base(filepath.FromSlash(p))
		return b == "GameAssembly.dll" || strings.HasSuffix(b, ".assets")
	}
	dest := t.TempDir()
	log := logx.New(logx.LevelError, false)

	st, err := ClientFiles(context.Background(), log, srv.URL+"/build", dest, Options{Filter: filter, Incremental: true})
	if err != nil {
		t.Fatalf("ClientFiles: %v", err)
	}
	if st.Total != 2 || st.Downloaded != 2 || st.Reused != 0 {
		t.Errorf("first run stats = %+v, want Total=2 Downloaded=2 Reused=0", st)
	}
	// Non-essential files must not have been downloaded.
	if _, err := os.Stat(filepath.Join(dest, "baselib.dll")); err == nil {
		t.Error("baselib.dll should have been filtered out")
	}
	if _, err := os.Stat(filepath.Join(dest, "RotMG Exalt_Data", "resources.resource")); err == nil {
		t.Error("resources.resource should have been filtered out")
	}

	// Second run: everything unchanged => all reused, nothing downloaded.
	st2, err := ClientFiles(context.Background(), log, srv.URL+"/build", dest, Options{Filter: filter, Incremental: true})
	if err != nil {
		t.Fatalf("ClientFiles run 2: %v", err)
	}
	if st2.Downloaded != 0 || st2.Reused != 2 {
		t.Errorf("second run stats = %+v, want Downloaded=0 Reused=2", st2)
	}
}

func TestPrune(t *testing.T) {
	dest := t.TempDir()
	// A stale file from a previous (larger) build.
	stale := filepath.Join(dest, "old", "stale.bin")
	os.MkdirAll(filepath.Dir(stale), 0o755)
	os.WriteFile(stale, []byte("old"), 0o644)

	files := map[string][]byte{"keep.bin": []byte("new")}
	srv := newCDN(t, files)
	defer srv.Close()

	st, err := ClientFiles(context.Background(), logx.New(logx.LevelError, false), srv.URL+"/build", dest, Options{Incremental: true})
	if err != nil {
		t.Fatalf("ClientFiles: %v", err)
	}
	if st.Pruned != 1 {
		t.Errorf("Pruned = %d, want 1", st.Pruned)
	}
	if _, err := os.Stat(stale); err == nil {
		t.Error("stale file should have been pruned")
	}
}

func TestSafeJoin(t *testing.T) {
	base := t.TempDir()
	if _, err := safeJoin(base, "RotMG Exalt_Data/level0"); err != nil {
		t.Errorf("valid path rejected: %v", err)
	}
	for _, bad := range []string{"../escape.txt", "a/../../escape", "../../etc/passwd"} {
		if _, err := safeJoin(base, bad); err == nil {
			t.Errorf("expected rejection for %q", bad)
		}
	}
}

func TestDownloadRetry(t *testing.T) {
	content := []byte("payload after retry")
	var gzHits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/checksum.json"):
			json.NewEncoder(w).Encode(Manifest{Files: []File{{Path: "f.bin", Checksum: md5hex(content)}}})
		case strings.HasSuffix(r.URL.Path, ".gz"):
			if atomic.AddInt32(&gzHits, 1) == 1 {
				http.Error(w, "transient", http.StatusInternalServerError) // fail first attempt
				return
			}
			w.Write(gzipped(content))
		default:
			http.Error(w, "nope", http.StatusForbidden)
		}
	}))
	defer srv.Close()

	dest := t.TempDir()
	st, err := ClientFiles(context.Background(), logx.New(logx.LevelError, false), srv.URL+"/build", dest, Options{})
	if err != nil {
		t.Fatalf("ClientFiles: %v", err)
	}
	if st.Downloaded != 1 {
		t.Errorf("Downloaded = %d, want 1", st.Downloaded)
	}
	got, _ := os.ReadFile(filepath.Join(dest, "f.bin"))
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch after retry")
	}
	if atomic.LoadInt32(&gzHits) < 2 {
		t.Errorf("expected a retry (gz hits = %d)", gzHits)
	}
}

func TestJoinURL(t *testing.T) {
	got := joinURL("https://cdn/build-release/h/id", "RotMG Exalt_Data/level0")
	want := "https://cdn/build-release/h/id/RotMG%20Exalt_Data/level0"
	if got != want {
		t.Errorf("joinURL = %q, want %q", got, want)
	}
}

func TestPlainThenGzFallback(t *testing.T) {
	// Plain path serves 200 directly (no gz needed).
	content := []byte("uncompressed file")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".gz") {
			http.Error(w, "no gz", http.StatusNotFound)
			return
		}
		w.Write(content)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "f.bin")
	ok, err := fetchTo(context.Background(), &http.Client{}, srv.URL+"/f.bin", false, dest, md5hex(content))
	if err != nil || !ok {
		t.Fatalf("plain fetch: ok=%v err=%v", ok, err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch")
	}
}
