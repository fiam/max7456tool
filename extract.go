package main

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"

	"github.com/fiam/max7456tool/mcm"

	"gopkg.in/urfave/cli.v1"
)

func extractAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("extract requires 2 arguments, see help extract")
	}
	f, err := os.Open(ctx.Args().Get(0))
	if err != nil {
		return err
	}
	defer f.Close()
	dec, err := mcm.NewDecoder(f)
	if err != nil {
		return err
	}
	dir := ctx.Args().Get(1)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	blanks := ctx.Bool("blanks")
	for ii := 0; ii < dec.NChars(); ii++ {
		ch := dec.CharAt(ii)
		if !blanks && ch.IsBlank() {
			continue
		}
		im := ch.Image(nil)
		output := filepath.Join(dir, fmt.Sprintf("%03d", ii)+".png")
		f, err := openOutputFile(output)
		if err != nil {
			return err
		}
		if err := png.Encode(f, im); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}
