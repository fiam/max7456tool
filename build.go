package main

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

func buildAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("build requires 2 arguments, see help build")
	}
	dir := ctx.Args().Get(0)
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	chars := make(map[int]*MCMChar)
	var builder charBuilder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if strings.ToLower(ext) == ".png" {
			nonExt := name[:len(name)-len(ext)]
			// Parse the name. It might contain multiple images
			lines := strings.Split(nonExt, "_")
			var nums [][]int
			w := 0
			h := len(lines)
			for _, line := range lines {
				var lineNums []int
				items := strings.Split(line, "-")
				if w != 0 && w != len(items) {
					return fmt.Errorf("uneven lines in filename %q: %d vs %d", nonExt, w, len(items))
				}
				w = len(items)
				for _, v := range items {
					n, err := strconv.Atoi(v)
					if err != nil {
						return fmt.Errorf("invalid number %q if image filename %q: %v", v, nonExt, err)
					}
					lineNums = append(lineNums, n)
				}
				nums = append(nums, lineNums)
			}
			// Decode the image
			filename := filepath.Join(dir, name)
			imf, err := os.Open(filename)
			if err != nil {
				return err
			}
			im, imfmt, err := image.Decode(imf)
			if err != nil {
				return fmt.Errorf("error decoding %s: %v", filename, err)
			}
			if err := imf.Close(); err != nil {
				return err
			}
			if imfmt != "png" {
				return fmt.Errorf("%s: invalid image format %s, must be png", filename, imfmt)
			}
			if im.Bounds().Dx() != w*charWidth {
				return fmt.Errorf("image with %d characters per line must have a width of %d, not %d", w, w*charWidth, im.Bounds().Dx())
			}
			if im.Bounds().Dy() != h*charHeight {
				return fmt.Errorf("image with %d characters per column must have a height of %d, not %d", h, h*charHeight, im.Bounds().Dy())
			}
			bounds := im.Bounds()
			// Import each character
			for jj := 0; jj < h; jj++ {
				for ii := 0; ii < w; ii++ {
					chNum := nums[jj][ii]
					x0 := bounds.Min.X + ii*charWidth
					y0 := bounds.Min.Y + jj*charHeight
					if debugFlag {
						log.Printf("importing char %d from image %v @%d,%d", chNum, name, x0, y0)
					}
					builder.Reset()
					for y := y0; y < y0+charHeight; y++ {
						for x := x0; x < x0+charWidth; x++ {
							r, g, b, a := im.At(x, y).RGBA()
							var p MCMPixel
							switch {
							case r == 0 && g == 0 && b == 0 && a == 65535:
								p = MCMPixelBlack
							case r == 65535 && g == 65535 && b == 65535 && a == 65535:
								p = MCMPixelWhite
							default:
								p = MCMPixelTransparent
							}
							builder.AppendPixel(p)
						}
					}
					for !builder.IsComplete() {
						builder.AppendPixel(MCMPixelTransparent)
					}
					if _, found := chars[chNum]; found {
						return fmt.Errorf("duplicate character %d", chNum)
					}
					chars[chNum] = builder.Char()
					builder.Reset()
				}
			}
		}
	}
	output := ctx.Args().Get(1)
	f, err := openOutputFile(output)
	if err != nil {
		return err
	}
	enc := &Encoder{
		Chars: chars,
		Fill:  !ctx.Bool("no-blanks"),
	}
	if err := enc.Encode(f); err != nil {
		// Remove the file, since it can't be
		// a proper .mcm at this point
		os.Remove(output)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}
