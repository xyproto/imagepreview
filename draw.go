package imagepreview

import (
	"image"
	"image/color"
	"os"

	"github.com/xyproto/palgen"
	"github.com/xyproto/vt"
	"golang.org/x/image/draw"
)

// BlockRune is the UTF-8 block character used for text-based image rendering.
const BlockRune = '▒'

// ASCIIRune is the ASCII fallback character used for text-based image rendering.
const ASCIIRune = '#'

// DrawOnCanvas draws the given image onto a VT100 Canvas using the basic 16-color palette.
// The drawRune parameter specifies the character used for each pixel
// (typically BlockRune or ASCIIRune).
func DrawOnCanvas(canvas *vt.Canvas, m image.Image, drawRune rune) error {
	img, err := palgen.ConvertBasic(m)
	if err != nil {
		return err
	}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			vc := vt.White // default
			if found, ok := PaletteColorMap[[3]uint8{c.R, c.G, c.B}]; ok {
				vc = found
			}
			canvas.PlotColor(uint(x), uint(y), vc, drawRune)
		}
	}
	return nil
}

// DrawTextImage renders an image file into a region of a VT100 Canvas using colored
// block characters. col and row specify the top-left corner (0-indexed canvas
// coordinates); cols and rows specify the available area in terminal cells.
func DrawTextImage(canvas *vt.Canvas, path string, col, row, cols, rows uint) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return
	}

	bounds := img.Bounds()
	imgW := float64(bounds.Dx())
	imgH := float64(bounds.Dy())
	if imgW == 0 || imgH == 0 {
		return
	}

	width := int(cols)
	height := int(rows)

	// Adjustment for terminal cell aspect ratio (roughly 2:1 height:width)
	ratio := (imgH / imgW) * 2.0

	if proportionalWidth := int(float64(height) / ratio); proportionalWidth < width {
		width = proportionalWidth
	} else if proportionalHeight := int(float64(width) * ratio); proportionalHeight < height {
		height = proportionalHeight
	}

	if width <= 0 || height <= 0 {
		return
	}

	resizedImage := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(resizedImage, resizedImage.Rect, img, bounds, draw.Over, nil)

	indexedImg, err := palgen.ConvertBasic(resizedImage)
	if err != nil {
		return
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c := color.NRGBAModel.Convert(indexedImg.At(x, y)).(color.NRGBA)
			vc := vt.White // default
			if found, ok := PaletteColorMap[[3]uint8{c.R, c.G, c.B}]; ok {
				vc = found
			}
			canvas.PlotColor(col+uint(x), row+uint(y), vc, BlockRune)
		}
	}
}
