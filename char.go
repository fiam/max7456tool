package main

import (
	"fmt"
	"image"
	"image/color"
)

const (
	charWidth  = 12
	charHeight = 18

	// See type MCMChar
	minCharBytes = 54
	charBytes    = 64
)

var (
	blankCharacter = constantChar(85) // all pixels = 01
)

// MCMPixel represents a pixel in a character. Each pixel must be one
// of MCMPixelBlack, MCMPixelTransparent or MCMPixelWhite.
type MCMPixel byte

const (
	// MCMPixelBlack represents a black pixel
	MCMPixelBlack MCMPixel = 0
	// MCMPixelTransparent represents a transparent/gray pixel,
	// depending on the OSD mode.
	MCMPixelTransparent = 1
	// MCMPixelWhite represents a white pixel
	MCMPixelWhite = 2
)

func (p MCMPixel) isTransparent() bool {
	// Transparent pixels have the LSB
	// set to 1 while MSB is ignored.
	return (p & MCMPixelTransparent) == MCMPixelTransparent
}

// MCMChar represents a character in the character map.
// Each character has 12x18 lines, where each pixel is represented
// by 2 bits. Thus, each character requires ((12*18)*2)/8 = 54 bytes.
// However, MCM files use 64 bytes per character ignoring the rest
// (according to Maxim, to make adressing easier).
type MCMChar struct {
	data []byte
}

// Data returns a copy of the raw pixel data.
func (c *MCMChar) Data() []byte {
	data := make([]byte, len(c.data))
	copy(data, c.data)
	return data
}

// ForEachPixel calls f for each pixel in the character.
// 0 <= x <= 12 while y >= 0. Note that a character might
// have extra ignored pixels at the end. unused will be true
// for those ones. p will always be one
// of the constants defined for MCMPixel
func (c *MCMChar) ForEachPixel(f func(x, y int, unused bool, p MCMPixel)) {
	x := 0
	y := 0
	for _, v := range c.data {
		unused := y >= charHeight
		f(x, y, unused, MCMPixel((0xC0&v)>>6))
		f(x+1, y, unused, MCMPixel((0x30&v)>>4))
		f(x+2, y, unused, MCMPixel((0xC&v)>>2))
		f(x+3, y, unused, MCMPixel(0x03&v))

		x += 4
		if x == charWidth {
			x = 0
			y++
		}
	}
}

// Image returns a 12x18 image of the character
func (c *MCMChar) Image(transparent color.Color) image.Image {
	if isNilColor(transparent) {
		transparent = defaultTransparentColor
	}
	im := image.NewRGBA(image.Rect(0, 0, charWidth, charHeight))
	c.ForEachPixel(func(x, y int, unused bool, p MCMPixel) {
		if unused {
			return
		}
		var c color.Color
		switch p {
		case MCMPixelTransparent:
			c = transparent
		case MCMPixelBlack:
			c = blackColor
		case MCMPixelWhite:
			c = whiteColor
		default:
			// Should not happen
			panic(fmt.Errorf("invalid pixel %v", p))
		}
		im.Set(x, y, c)
	})
	return im
}

func (c *MCMChar) isBlank() bool {
	blank := true
	c.ForEachPixel(func(x, y int, unused bool, p MCMPixel) {
		if p != MCMPixelTransparent {
			blank = false
		}
	})
	return blank
}

func constantChar(b byte) *MCMChar {
	data := make([]byte, charBytes)
	for ii := range data {
		data[ii] = b
	}
	return &MCMChar{data: data}
}

type charBuilder struct {
	c     *MCMChar
	pixel int
}

func (b *charBuilder) Char() *MCMChar {
	return b.c
}

func (b *charBuilder) Reset() {
	b.c = new(MCMChar)
	b.pixel = 0
}

func (b *charBuilder) IsEmpty() bool {
	return len(b.c.data) == 0
}

func (b *charBuilder) IsComplete() bool {
	return len(b.c.data) == charBytes && b.pixel == 0
}

func (b *charBuilder) AppendPixel(p MCMPixel) error {
	if p.isTransparent() {
		p = MCMPixelTransparent
	}
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
