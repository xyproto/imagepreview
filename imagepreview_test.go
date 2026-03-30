package imagepreview

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// testPNG writes a small solid-color PNG to dir and returns its path.
func testPNG(t *testing.T, dir, name string, w, h int, c color.Color) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestIsImageExt(t *testing.T) {
	yes := []string{
		"photo.png", "photo.PNG", "photo.jpg", "photo.jpeg",
		"photo.gif", "photo.svg", "photo.bmp", "photo.webp",
		"photo.qoi", "photo.ico", "photo.jxl",
	}
	no := []string{"readme.md", "main.go", ""}

	for _, p := range yes {
		if !IsImageExt(p) {
			t.Errorf("IsImageExt(%q) = false, want true", p)
		}
	}
	for _, p := range no {
		if IsImageExt(p) {
			t.Errorf("IsImageExt(%q) = true, want false", p)
		}
	}
}

func TestLoadImagePNG(t *testing.T) {
	dir := t.TempDir()
	path := testPNG(t, dir, "red.png", 4, 4, color.RGBA{255, 0, 0, 255})

	nrgba, err := LoadImage(path)
	if err != nil {
		t.Fatal(err)
	}
	if nrgba.Bounds().Dx() != 4 || nrgba.Bounds().Dy() != 4 {
		t.Fatalf("got %dx%d, want 4x4", nrgba.Bounds().Dx(), nrgba.Bounds().Dy())
	}
	r, g, b, _ := nrgba.At(0, 0).RGBA()
	if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
		t.Errorf("pixel (0,0): got (%d,%d,%d), want (255,0,0)", r>>8, g>>8, b>>8)
	}
}

func TestConvertToNRGBA(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{0, 255, 0, 255})
	src.Set(1, 1, color.RGBA{0, 0, 255, 255})

	nrgba, err := ConvertToNRGBA(src)
	if err != nil {
		t.Fatal(err)
	}
	if nrgba.Bounds() != src.Bounds() {
		t.Fatalf("bounds: got %v, want %v", nrgba.Bounds(), src.Bounds())
	}
	_, g, _, _ := nrgba.At(0, 0).RGBA()
	if g>>8 != 255 {
		t.Errorf("pixel (0,0): green channel got %d, want 255", g>>8)
	}
}

func TestScaleNearestNeighbor(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	src.Set(0, 0, color.RGBA{255, 0, 0, 255})
	src.Set(1, 0, color.RGBA{0, 255, 0, 255})
	src.Set(0, 1, color.RGBA{0, 0, 255, 255})
	src.Set(1, 1, color.RGBA{255, 255, 0, 255})

	dst := ScaleNearestNeighbor(src, 4, 4)
	if dst.Bounds().Dx() != 4 || dst.Bounds().Dy() != 4 {
		t.Fatalf("got %dx%d, want 4x4", dst.Bounds().Dx(), dst.Bounds().Dy())
	}
	// The top-left 2x2 block should all be red (from src pixel 0,0).
	for y := range 2 {
		for x := range 2 {
			r, g, b, _ := dst.At(x, y).RGBA()
			if r>>8 != 255 || g>>8 != 0 || b>>8 != 0 {
				t.Errorf("pixel (%d,%d): got (%d,%d,%d), want red", x, y, r>>8, g>>8, b>>8)
			}
		}
	}
}

func TestAspectRatioCells(t *testing.T) {
	// Square image in a square pane (TerminalCellPixels returns at least 8x16).
	cols, rows := AspectRatioCells(100, 100, 20, 20)
	if cols == 0 || rows == 0 {
		t.Fatalf("got %dx%d, want non-zero", cols, rows)
	}
	if cols > 20 || rows > 20 {
		t.Errorf("exceeds available area: %dx%d", cols, rows)
	}

	// Zero image dimensions must return the available area unchanged.
	cols, rows = AspectRatioCells(0, 0, 10, 10)
	if cols != 10 || rows != 10 {
		t.Errorf("zero image: got %dx%d, want 10x10", cols, rows)
	}
}

func TestLoadAndEncode(t *testing.T) {
	dir := t.TempDir()
	path := testPNG(t, dir, "green.png", 16, 16, color.RGBA{0, 255, 0, 255})

	result, err := LoadAndEncode(context.Background(), path, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != path {
		t.Errorf("path: got %q, want %q", result.Path, path)
	}
	if result.Encoded == "" {
		t.Error("expected non-empty encoded data")
	}
	if result.ImgW < 16 || result.ImgH < 16 {
		t.Errorf("dimensions: got %dx%d, want at least 16x16", result.ImgW, result.ImgH)
	}
}

func TestLoadAndEncodeCancelled(t *testing.T) {
	dir := t.TempDir()
	path := testPNG(t, dir, "cancel.png", 4, 4, color.RGBA{0, 0, 0, 255})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before the call

	_, err := LoadAndEncode(ctx, path, 800, 600)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

func TestFlushImageWritesOutput(t *testing.T) {
	// Create a small encoded PNG to exercise FlushImage.
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	// Use LoadAndEncode to get a real base64 string.
	dir := t.TempDir()
	path := testPNG(t, dir, "tiny.png", 2, 2, color.White)
	result, err := LoadAndEncode(context.Background(), path, 400, 400)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	FlushImage(&out, result.Encoded, 10, 10)
	if out.Len() == 0 {
		t.Error("FlushImage produced no output")
	}
}

func TestPaletteColorMapPopulated(t *testing.T) {
	if len(PaletteColorMap) != 16 {
		t.Errorf("PaletteColorMap: got %d entries, want 16", len(PaletteColorMap))
	}
}

func TestTerminalCellPixels(t *testing.T) {
	cellW, cellH := TerminalCellPixels()
	if cellW == 0 || cellH == 0 {
		t.Errorf("got %dx%d, want positive dimensions", cellW, cellH)
	}
}
