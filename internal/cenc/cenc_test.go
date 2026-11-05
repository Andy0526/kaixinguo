package cenc

import (
	"bytes"
	"encoding/hex"
	"math/rand"
	"os"
	"strings"
	"testing"
)

// TestSample decrypts a real episode when CENC_SAMPLE names one and
// CENC_KEY holds its key: whole, and again in random pieces, which must
// agree byte for byte, as a player seeking through the stream needs.
// CENC_OUT, when set, receives the clear file.
func TestSample(t *testing.T) {
	path, keyHex := os.Getenv("CENC_SAMPLE"), strings.TrimSpace(os.Getenv("CENC_KEY"))
	if path == "" || keyHex == "" {
		t.Skip("CENC_SAMPLE and CENC_KEY are not set")
	}
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	key, err := hex.DecodeString(keyHex)
	if err != nil {
		t.Fatal(err)
	}
	fetch := func(off, n int64) ([]byte, error) { return src[off : off+n], nil }
	l, err := Open(fetch, int64(len(src)), key)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Protected() {
		t.Fatal("no protected samples")
	}
	whole := append([]byte(nil), src...)
	l.Decrypt(whole, 0)
	if out := os.Getenv("CENC_OUT"); out != "" {
		if err := os.WriteFile(out, whole, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := rand.New(rand.NewSource(1))
	for round := 0; round < 20; round++ {
		pieces := append([]byte(nil), src...)
		for off := 0; off < len(pieces); {
			n := min(1+r.Intn(70000), len(pieces)-off)
			l.Decrypt(pieces[off:off+n], int64(off))
			off += n
		}
		if !bytes.Equal(pieces, whole) {
			t.Fatalf("round %d: pieces differ from the whole", round)
		}
	}
}
