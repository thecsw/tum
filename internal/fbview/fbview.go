// Package fbview captures and analyzes the live qtfb framebuffer from the
// reMarkable, so you can verify app rendering without eyes on the e-ink screen.
//
// The qtfb-shim exposes its virtual framebuffer as a shared-memory file at
// /dev/shm/qtfb_<key>. tum grabs the newest one over SSH, then:
//   - analyzes orientation (portrait vs landscape-rotated text)
//   - writes a PNG (optionally rotated for landscape viewing)
package fbview

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
)

// Framebuffer dimensions (qtfb RM2FB mode: RGB565 1404x1872).
const (
	FbWidth  = 1404
	FbHeight = 1872
	BPP      = 2
)

// RGB565 pixel: little-endian uint16.
type RGB565 uint16

func (p RGB565) RGBA() (r, g, b, a uint32) {
	r5 := (uint16(p) >> 11) & 0x1f
	g6 := (uint16(p) >> 5) & 0x3f
	b5 := uint16(p) & 0x1f
	return uint32(r5)<<3 | uint32(r5)>>2, uint32(g6)<<2 | uint32(g6)>>4, uint32(b5)<<3 | uint32(b5)>>2, 0xffff
}

// Analysis describes the framebuffer content.
type Analysis struct {
	Width, Height int
	ContentBBox   [4]int // minx, maxx, miny, maxy
	HorizBands    int    // horizontal text lines (rows with dark pixels)
	VertBands     int    // vertical text lines (cols with dark pixels)
	Orientation   string // "portrait", "landscape-rotated", "empty", "ambiguous"
}

// Analyze decodes the RGB565 framebuffer bytes and detects text orientation.
// Portrait text has many horizontal bands (rows); rotated text has vertical
// bands (columns).
func Analyze(data []byte) Analysis {
	a := Analysis{Width: FbWidth, Height: FbHeight}
	if len(data) < FbWidth*FbHeight*BPP {
		a.Orientation = "too-small"
		return a
	}

	rowDark := make([]int, FbHeight)
	colDark := make([]int, FbWidth)
	minx, maxx, miny, maxy := -1, -1, -1, -1

	// Sample every other pixel for speed.
	for y := 0; y < FbHeight; y += 2 {
		base := y * FbWidth * BPP
		for x := 0; x < FbWidth; x += 2 {
			val := binary.LittleEndian.Uint16(data[base+x*BPP:])
			r := (val >> 11) & 0x1f
			g := (val >> 5) & 0x3f
			b := val & 0x1f
			if (r+g+b)/3 < 25 {
				rowDark[y]++
				colDark[x]++
				if minx == -1 || x < minx {
					minx = x
				}
				if x > maxx {
					maxx = x
				}
				if miny == -1 || y < miny {
					miny = y
				}
				if y > maxy {
					maxy = y
				}
			}
		}
	}

	a.ContentBBox = [4]int{minx, maxx, miny, maxy}
	a.HorizBands = countBands(rowDark, 4)
	a.VertBands = countBands(colDark, 4)

	if minx == -1 {
		a.Orientation = "empty"
	} else if a.HorizBands > a.VertBands*2 {
		a.Orientation = "portrait"
	} else if a.VertBands > a.HorizBands*2 {
		a.Orientation = "landscape-rotated"
	} else {
		a.Orientation = "ambiguous"
	}
	return a
}

func countBands(arr []int, thresh int) int {
	bands, inBand := 0, false
	for _, v := range arr {
		if v > thresh && !inBand {
			bands++
			inBand = true
		} else if v <= thresh {
			inBand = false
		}
	}
	return bands
}

// String formats the analysis for display.
func (a Analysis) String() string {
	if a.Orientation == "too-small" {
		return fmt.Sprintf("framebuffer too small (%d bytes)", 0)
	}
	if a.Orientation == "empty" {
		return fmt.Sprintf("%dx%d, content: EMPTY", a.Width, a.Height)
	}
	bb := a.ContentBBox
	return fmt.Sprintf("%dx%d, content bbox %dx%d at (%d,%d), h-bands=%d v-bands=%d → %s",
		a.Width, a.Height, bb[1]-bb[0], bb[3]-bb[2], bb[0], bb[2], a.HorizBands, a.VertBands, a.Orientation)
}

// ToImage converts RGB565 bytes to an image.RGBA.
func ToImage(data []byte) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, FbWidth, FbHeight))
	for y := 0; y < FbHeight; y++ {
		base := y * FbWidth * BPP
		for x := 0; x < FbWidth; x++ {
			val := binary.LittleEndian.Uint16(data[base+x*BPP:])
			r5 := (val >> 11) & 0x1f
			g6 := (val >> 5) & 0x3f
			b5 := val & 0x1f
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(r5<<3 | r5>>2),
				G: uint8(g6<<2 | g6>>4),
				B: uint8(b5<<3 | b5>>2),
				A: 0xff,
			})
		}
	}
	return img
}

// WritePNG writes the framebuffer as a PNG, optionally rotated 90° CW.
func WritePNG(data []byte, path string, rotate90 bool) error {
	img := ToImage(data)
	if rotate90 {
		rotated := image.NewRGBA(image.Rect(0, 0, FbHeight, FbWidth))
		for y := 0; y < FbHeight; y++ {
			for x := 0; x < FbWidth; x++ {
				rotated.Set(FbHeight-1-y, x, img.At(x, y))
			}
		}
		img = rotated
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
