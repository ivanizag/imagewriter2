package main

import (
	"image"
	"image/color"
	"image/gif"
	"os"
)

/*
The paper keeps the dots of the graphics where the print head puts them, to
be saved as a picture. The head moves across in a column of the pitch, and
the paper moves up in 1/144 of an inch, the unit of ESC T. A dot is 1/72 of an
inch, two units, the distance between the pins.

The picture has 144 pixels per inch both ways, so it has the shape of the
paper, and a dot is 2 by 2 pixels. Text is not drawn, only the graphics.
*/

const (
	pixelsPerInch = 144
	dotPixels     = 2
	paperInches   = 8 // The width of a line of the ImageWriter II
)

// The bands of the colour ribbon, each a bit
const (
	bandBlack = 1 << iota
	bandYellow
	bandRed
	bandBlue
)

// colorBands are the bands of the ribbon for each color of ESC K: black,
// yellow, red, blue, and orange, green and purple printed with two bands
var colorBands = []uint8{
	bandBlack,
	bandYellow,
	bandRed,
	bandBlue,
	bandYellow | bandRed,
	bandYellow | bandBlue,
	bandRed | bandBlue,
}

// dotsPerInch of the graphics for each pitch, the dots of a character are 8
var dotsPerInch = map[uint8]float64{
	'n': 72,
	'N': 80,
	'E': 96,
	'e': 107.2,
	'q': 120,
	'Q': 136,
	'p': 144,
	'P': 160,
}

type paper struct {
	x            float64 // in inches from the left
	y            int     // in 1/144 inches from the top
	dpi          float64
	lineDistance int // in 1/144 inches
	pageLength   int // in 1/144 inches
	reverse      bool
	bands        uint8
	width        int
	height       int
	dots         [][]uint8 // a row of pixels each, the bands printed on them
}

func newPaper() *paper {
	return &paper{
		dpi:          dotsPerInch['E'],
		lineDistance: 24,
		pageLength:   11 * 144,
		bands:        bandBlack,
		width:        paperInches * pixelsPerInch,
	}
}

func (p *paper) setPitch(command uint8) {
	p.dpi = dotsPerInch[command]
}

func (p *paper) setColor(color int) {
	if color >= 0 && color < len(colorBands) {
		p.bands = colorBands[color]
	}
}

func (p *paper) setHeadPosition(dots int) {
	p.x = float64(dots) / p.dpi
}

func (p *paper) carriageReturn() {
	p.x = 0
}

func (p *paper) lineFeed() {
	if p.reverse {
		p.y -= p.lineDistance
		if p.y < 0 {
			p.y = 0
		}
	} else {
		p.y += p.lineDistance
	}
}

func (p *paper) formFeed() {
	p.y = (p.y/p.pageLength + 1) * p.pageLength
}

// graphics prints a line of graphics, a column of 8 dots for each byte, the
// least significant bit at the top
func (p *paper) graphics(columns []uint8) {
	for _, column := range columns {
		left := int(p.x * pixelsPerInch)
		for bit := 0; bit < 8; bit++ {
			if column&(1<<bit) != 0 {
				p.dot(left, p.y+bit*dotPixels)
			}
		}
		p.x += 1 / p.dpi
	}
}

func (p *paper) dot(left int, top int) {
	for len(p.dots) < top+dotPixels {
		p.dots = append(p.dots, nil)
	}
	for y := top; y < top+dotPixels; y++ {
		for len(p.dots[y]) < left+dotPixels {
			p.dots[y] = append(p.dots[y], 0)
		}
		for x := left; x < left+dotPixels; x++ {
			p.dots[y][x] |= p.bands
		}
	}
	if left+dotPixels > p.width {
		p.width = left + dotPixels
	}
	if top+dotPixels > p.height {
		p.height = top + dotPixels
	}
}

// ink is the color of the paper with the bands printed on it, each band
// taking away the light it absorbs
func ink(bands uint8) color.RGBA {
	if bands&bandBlack != 0 {
		return color.RGBA{0, 0, 0, 255}
	}
	c := color.RGBA{255, 255, 255, 255}
	if bands&bandYellow != 0 {
		c.B = 0
	}
	if bands&bandRed != 0 {
		c.G = 0
	}
	if bands&bandBlue != 0 {
		c.R = 0
	}
	return c
}

func (p *paper) image() *image.Paletted {
	palette := make(color.Palette, 16)
	for bands := range palette {
		palette[bands] = ink(uint8(bands))
	}
	img := image.NewPaletted(image.Rect(0, 0, p.width, p.height), palette)
	for y, row := range p.dots {
		for x, bands := range row {
			img.SetColorIndex(x, y, bands)
		}
	}
	return img
}

func (p *paper) saveGif(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	return gif.Encode(f, p.image(), nil)
}
