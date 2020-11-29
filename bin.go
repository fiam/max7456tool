package main

import (
	"bytes"
	"errors"
	"os"

	"github.com/fiam/max7456tool/mcm"

	cli "github.com/urfave/cli/v2"
)

const (
	flipHorizontalPixelsFlagName = "flip-horizontal-pixels"
)

func flipHorizontalBytePixels(c byte) byte {
	return (c >> 6) | (c << 6) | ((c >> 2) & (3 << 2)) | ((c << 2) & (3 << 4))
}

func buildBinFromMCM(ctx *cli.Context, output string, input string, flipHorizontalPixels bool) error {
	mf, err := os.Open(input)
	if err != nil {
		return err
	}
	defer mf.Close()
	dec, err := mcm.NewDecoder(mf)
	if err != nil {
		return err
	}
	// Write all data to a buffer
	var buf bytes.Buffer
	for ii := 0; ii < dec.NChars(); ii++ {
		chr := dec.CharAt(ii)
		data := chr.Data()
		if flipHorizontalPixels {
			for jj := 0; jj < mcm.MinCharBytes; jj++ {
				data[jj] = flipHorizontalBytePixels(data[jj])
			}
		}
		if _, err := buf.Write(data); err != nil {
			return err
		}
	}
	// Save to bin
	f, err := openOutputFile(output)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func binAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("bin requires 2 arguments, see help bin")
	}
	input := ctx.Args().Get(0)
	output := ctx.Args().Get(1)
	flipHorizontalPixels := ctx.Bool(flipHorizontalPixelsFlagName)
	return buildBinFromMCM(ctx, output, input, flipHorizontalPixels)
}
