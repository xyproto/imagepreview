package imagepreview

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndEncodeSVG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.svg")
	svgData := `<svg width="100" height="100"><rect x="0" y="0" width="100" height="100" fill="red" /></svg>`
	if err := os.WriteFile(path, []byte(svgData), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := LoadAndEncode(context.Background(), path, 200, 200)
	if err != nil {
		t.Fatal(err)
	}
	if result.ImgW != 200 {
		t.Errorf("got width %d, want 200", result.ImgW)
	}
	if result.ImgH != 200 {
		t.Errorf("got height %d, want 200", result.ImgH)
	}
	if result.Encoded == "" {
		t.Error("expected non-empty encoded data")
	}
}
