package mcm

import (
	"fmt"
	"image"
	"image/color"
)

const (
	// CharWidth is the width of a character in pixels
	CharWidth = 12
	// CharHeight is the height of a character in pixels
	CharHeight = 18

	// MinCharBytes is the minimum amount of bytes present
	// in a character
	MinCharBytes = 54
	// CharBytes is the default character size used in
	// .mcm files.
	CharBytes = 64

	// CharNum is the default number of characters in an MCM file
	CharNum = 256
	// ExtendedCharNum is the number of characters in an MCM file
	// for FrSkyOSD or AT7456, which support 2 pages of characters
	ExtendedCharNum = 512

	// all pixels = 01
	mcmTransparentByte = 85
)

var (
	blankCharacter = constantChar(mcmTransparentByte)
)

// Pixel represents a pixel in a character. Each pixel must be one
// of PixelBlack, PixelTransparent or PixelWhite.
type Pixel byte

const (
	// PixelBlack represents a black pixel
	PixelBlack Pixel = 0
	// PixelTransparent represents a transparent/gray pixel,
	// depending on the OSD mode.
	PixelTransparent = 1
	// PixelWhite represents a white pixel
	PixelWhite = 2
)

func (p Pixel) isTransparent() bool {
	// Transparent pixels have the LSB
	// set to 1 while MSB is ignored.
	return (p & PixelTransparent) == PixelTransparent
}

// Char represents a character in the character map.
// Each character has 12x18 lines, where each pixel is represented
// by 2 bits. Thus, each character requires ((12*18)*2)/8 = 54 bytes.
// However, MCM files use 64 bytes per character ignoring the rest
// (according to Maxim, to make adressing easier).
type Char struct {
	data []byte
}

// NewCharFromData returns a Char from its raw pixel data
func NewCharFromData(data []byte) (*Char, error) {
	if len(data) != CharBytes {
		return nil, fmt.Errorf("invalid char data size %d, must be %d", len(data), CharBytes)
	}
	return &Char{data: data}, nil
}

// NewCharFromImage returns a Char from an image, taking 12x18 pixels
// starting at (x0, y0).
func NewCharFromImage(im image.Image, x0 int, y0 int) (*Char, error) {
	var builder charBuilder
	if err := builder.SetImage(im, x0, y0); err != nil {
		return nil, err
	}
	return builder.Char(), nil
}

// Data returns a copy of the raw pixel data.
func (c *Char) Data() []byte {
	data := make([]byte, len(c.data))
	copy(data, c.data)
	return data
}

// ForEachPixel calls f for each pixel in the character.
// 0 <= x <= 12 while y >= 0. Note that a character might
// have extra ignored pixels at the end. unused will be true
// for those ones. p will always be one
// of the constants defined for Pixel
func (c *Char) ForEachPixel(f func(x, y int, unused bool, p Pixel)) {
	x := 0
	y := 0
	for _, v := range c.data {
		unused := y >= CharHeight
		f(x, y, unused, Pixel((0xC0&v)>>6))
		f(x+1, y, unused, Pixel((0x30&v)>>4))
		f(x+2, y, unused, Pixel((0xC&v)>>2))
		f(x+3, y, unused, Pixel(0x03&v))

		x += 4
		if x == CharWidth {
			x = 0
			y++
		}
	}
}

// ImageStrict returns a 12x18 image of the character. If the image contains
// any transparent pixels and no transparent color has been provided, an
// error will be returned. Same principle is applied for the undefined
// color. See Char.Image() for a simpler alternative.
func (c *Char) ImageStrict(transparent color.Color, undefined color.Color) (image.Image, error) {
	im := image.NewRGBA(image.Rect(0, 0, CharWidth, CharHeight))
	var err error
	c.ForEachPixel(func(x, y int, unused bool, p Pixel) {
		if err != nil {
			return
		}
		if unused {
			return
		}
		var c color.Color
		switch p {
		case PixelTransparent:
			if isNilColor(transparent) {
				err = fmt.Errorf("no color was provided for transparent pixel %v @ (%v, %v)", p, x, y)
				return
			}
			c = transparent
		case PixelBlack:
			c = BlackColor
		case PixelWhite:
			c = WhiteColor
		default:
			if isNilColor(undefined) {
				err = fmt.Errorf("no color was provided for undefined pixel %v @ (%v, %v)", p, x, y)
				return
			}
			c = undefined
		}
		im.Set(x, y, c)
	})
	if err != nil {
		return nil, err
	}
	return im, nil
}

// Image returns a 12x18 image of the character. Pixels with the
// 3 value (out of spec but usually considered as transparent) are
// considered as transparent. For more fine grained behavior, check
// Char.ImageStrict. If no transparent color is provided,
// DefaultTransparentColor will be used for transparent pixels.
func (c *Char) Image(transparent color.Color) image.Image {
	if isNilColor(transparent) {
		transparent = DefaultTransparentColor
	}
	im, err := c.ImageStrict(transparent, transparent)
	if err != nil {
		// Should not happen
		panic(err)
	}
	return im
}

// IsBlank returns true iff all pixels in the characters are
// transparent.
func (c *Char) IsBlank() bool {
	blank := true
	c.ForEachPixel(func(x, y int, unused bool, p Pixel) {
		if p != PixelTransparent {
			blank = false
		}
	})
	return blank
}

func constantChar(b byte) *Char {
	data := make([]byte, CharBytes)
	for ii := range data {
		data[ii] = b
	}
	return &Char{data: data}
}
