package mcm

import (
	"fmt"
	"image"
)

type charBuilder struct {
	c     *Char
	pixel int
}

func (b *charBuilder) Char() *Char {
	return b.c
}

func (b *charBuilder) Reset() {
	b.c = new(Char)
	b.pixel = 0
}

func (b *charBuilder) IsEmpty() bool {
	return len(b.c.data) == 0
}

func (b *charBuilder) IsComplete() bool {
	return len(b.c.data) == CharBytes && b.pixel == 0
}

func (b *charBuilder) AppendPixel(p Pixel) error {
	// Don't rewrite pixels. That would be problematic for OSDs
	// that do understand 11 as gray and/or use the metadata
	// (e.g. FrSkyOSD)
	if p > 3 {
		return fmt.Errorf("invalid pixel %v > 3", p)
	}
	pb := byte(p)
	switch b.pixel {
	case 0:
		// Append new byte
		b.c.data = append(b.c.data, pb<<6)
	case 1:
		b.c.data[len(b.c.data)-1] |= pb << 4
	case 2:
		b.c.data[len(b.c.data)-1] |= pb << 2
	case 3:
		b.c.data[len(b.c.data)-1] |= pb
	}
	b.pixel++
	if b.pixel == 4 {
		b.pixel = 0
	}
	return nil
}

func (b *charBuilder) SetImage(im image.Image, x0, y0 int) error {
	b.Reset()
	bounds := im.Bounds()
	for y := y0; y < y0+CharHeight; y++ {
		for x := x0; x < x0+CharWidth; x++ {
			px := bounds.Min.X + x
			py := bounds.Min.Y + y
			r, g, bl, a := im.At(px, py).RGBA()
			var p Pixel
			switch {
			case r == 0 && g == 0 && bl == 0 && a == 65535:
				p = PixelBlack
			case r == 65535 && g == 65535 && bl == 65535 && a == 65535:
				p = PixelWhite
			default:
				p = PixelTransparent
			}
			b.AppendPixel(p)
		}
	}
	for !b.IsComplete() {
		b.AppendPixel(PixelTransparent)
	}
	return nil
}
