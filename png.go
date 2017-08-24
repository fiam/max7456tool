package main

import (
	"errors"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"

	"gopkg.in/urfave/cli.v1"
)

func pngAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("png requires 2 arguments, see help png")
	}
	mf, err := os.Open(ctx.Args().Get(0))
	if err != nil {
		return err
	}
	defer mf.Close()
	dec, err := NewDecoder(mf)
	if err != nil {
		return err
	}
	cols := ctx.Int("columns")
	margin := ctx.Int("margin")
	rows := int(math.Ceil(float64(dec.NChars()) / float64(cols)))
	imageWidth := (charWidth+margin)*cols + margin
	imageHeight := (charHeight+margin)*rows + margin

	img := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))

	// Top and left sides of the grid
	for x := 0; x < imageWidth; x++ {
		for y := 0; y < margin; y++ {
			img.Set(x, y, blackColor)
		}
	}

	for x := 0; x < margin; x++ {
		for y := 0; y < imageHeight; y++ {
			img.Set(x, y, blackColor)
		}
	}

	// Draw each character
	for ii := 0; ii < cols; ii++ {
		for jj := 0; jj < rows; jj++ {
			leftX := ii*(charWidth+margin) + margin
			rightX := leftX + charWidth + margin
			topY := jj*(charHeight+margin) + margin
			bottomY := topY + charHeight + margin
			// Draw right line
			for x := rightX - margin; x < rightX; x++ {
				for y := topY; y < bottomY; y++ {
					img.Set(x, y, blackColor)
				}
			}
			// Draw bottom line
			for x := leftX; x < rightX; x++ {
				for y := bottomY - margin; y < bottomY; y++ {
					img.Set(x, y, blackColor)
				}
			}
			// Draw character
			chn := (jj * cols) + ii
			if chn >= dec.NChars() {
				continue
			}
			r := image.Rect(leftX, topY, leftX+charWidth, topY+charHeight)
			ch := dec.CharAt(chn)
			cim := ch.Image(nil)
			draw.Draw(img, r, cim, image.ZP, draw.Over)
		}
	}

	// Save to png
	f, err := openOutputFile(ctx.Args().Get(1))
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
