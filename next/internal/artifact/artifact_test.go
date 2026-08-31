package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPutGetRoundTrip(t *testing.T) {
	s := Open(t.TempDir())
	body := []byte(`{"receipt": true}`)
	digest, err := s.Put(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(digest) != 64 || digest != strings.ToLower(digest) {
		t.Fatalf("digest %q is not lowercase-hex sha256", digest)
	}
	got, err := s.Get(digest)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("round trip lost bytes: %q", got)
	}
	if Digest(body) != digest {
		t.Fatalf("Digest disagrees with Put")
	}
}

func TestGetRefusesCorruptContent(t *testing.T) {
	dir := t.TempDir()
	s := Open(dir)
	digest, err := s.Put([]byte("original"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sha256", digest), []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get(digest); err == nil || !strings.Contains(err.Error(), "corrupt") {
		t.Fatalf("corrupt-on-disk content must refuse naming corruption, got %v", err)
	}
}

func TestGetRefusesBadDigestForm(t *testing.T) {
	s := Open(t.TempDir())
	for _, bad := range []string{"", "abc", strings.Repeat("A", 64), "../escape"} {
		if _, err := s.Get(bad); err == nil {
			t.Fatalf("digest %q must refuse", bad)
		}
	}
}

func TestConcurrentPutsOfSameContent(t *testing.T) {
	s := Open(t.TempDir())
	body := []byte(strings.Repeat("x", 4096))
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Put(body); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if _, err := s.Get(Digest(body)); err != nil {
		t.Fatal(err)
	}
}
