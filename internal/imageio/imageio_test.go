package imageio

import (
	"image/png"
	"os"
	"runtime"
	"testing"
)

// TestHEIC decodes the HEIC file IMAGEIO_SAMPLE names, on macOS.
func TestHEIC(t *testing.T) {
	path := os.Getenv("IMAGEIO_SAMPLE")
	if path == "" || runtime.GOOS != "darwin" {
		t.Skip("IMAGEIO_SAMPLE is not set")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	img, err := Decode(data, 200)
	if err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("IMAGEIO_OUT"); out != "" {
		f, err := os.Create(out)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}
	b := img.Bounds()
	if b.Dx() != 200 || b.Dy() < 100 {
		t.Fatalf("size %v", b)
	}
	// A cover is not one flat color.
	first := img.RGBAAt(0, 0)
	same := true
	for y := 0; y < b.Dy() && same; y += 7 {
		for x := 0; x < b.Dx(); x += 7 {
			if img.RGBAAt(x, y) != first {
				same = false
				break
			}
		}
	}
	if same {
		t.Fatal("decoded a flat picture")
	}
}
