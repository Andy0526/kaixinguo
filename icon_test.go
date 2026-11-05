package main

import (
	"image/png"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestIcon renders resources/icon.svg into resources/icon.png when
// KX_ICON is set: go generate's job, kept as a test so it needs no tool.
func TestIcon(t *testing.T) {
	if os.Getenv("KX_ICON") == "" {
		t.Skip("KX_ICON is not set")
	}
	data, err := os.ReadFile("resources/icon.svg")
	if err != nil {
		t.Fatal(err)
	}
	svg, err := ui.ParseSVG(data)
	if err != nil {
		t.Fatal(err)
	}
	tt := ui.NewTester(func(c *ui.Context) {
		c.Root().Background(ui.Transparent)
		ui.Image(c, svg).Size(1024, 1024)
	}, 1024, 1024)
	img := tt.Image()
	if b := img.Bounds(); b.Dx() != 1024 {
		t.Fatalf("rendered %v", b)
	}
	f, err := os.Create("resources/icon.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
