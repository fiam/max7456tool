package main

import (
	"errors"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/fiam/max7456tool/mcm"

	"github.com/urfave/cli/v2"
)

func buildPNGFromMCM(ctx *cli.Context, output string, input string) error {
	mf, err := os.Open(input)
	if err != nil {
		return err
	}
	defer mf.Close()
	dec, err := mcm.NewDecoder(mf)
	if err != nil {
		return err
	}
	cols := ctx.Int("columns")
	margin := ctx.Int("margin")
	rows := int(math.Ceil(float64(dec.NChars()) / float64(cols)))
	imageWidth := (mcm.CharWidth+margin)*cols + margin
	imageHeight := (mcm.CharHeight+margin)*rows + margin

	img := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))

	// Top and left sides of the grid
	for x := 0; x < imageWidth; x++ {
		for y := 0; y < margin; y++ {
			img.Set(x, y, mcm.BlackColor)
		}
	}

	for x := 0; x < margin; x++ {
		for y := 0; y < imageHeight; y++ {
			img.Set(x, y, mcm.BlackColor)
		}
	}

	// Draw each character
	for ii := 0; ii < cols; ii++ {
		for jj := 0; jj < rows; jj++ {
			leftX := ii*(mcm.CharWidth+margin) + margin
			rightX := leftX + mcm.CharWidth + margin
			topY := jj*(mcm.CharHeight+margin) + margin
			bottomY := topY + mcm.CharHeight + margin
			// Draw right line
			for x := rightX - margin; x < rightX; x++ {
				for y := topY; y < bottomY; y++ {
					img.Set(x, y, mcm.BlackColor)
				}
			}
			// Draw bottom line
			for x := leftX; x < rightX; x++ {
				for y := bottomY - margin; y < bottomY; y++ {
					img.Set(x, y, mcm.BlackColor)
				}
			}
			// Draw character
			chn := (jj * cols) + ii
			if chn >= dec.NChars() {
				continue
			}
			r := image.Rect(leftX, topY, leftX+mcm.CharWidth, topY+mcm.CharHeight)
			ch := dec.CharAt(chn)
			cim := ch.Image(nil)
			draw.Draw(img, r, cim, image.ZP, draw.Over)
		}
	}

	// Save to png
	f, err := openOutputFile(output)
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

func pngAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("png requires 2 arguments, see help png")
	}
	input := ctx.Args().Get(0)
	output := ctx.Args().Get(1)
	return buildPNGFromMCM(ctx, output, input)
}
