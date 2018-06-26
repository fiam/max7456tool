package main

import (
	"errors"
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/urfave/cli.v1"
)

func buildMCM(ctx *cli.Context, chars map[int]*MCMChar) error {
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

func buildFromDirAction(ctx *cli.Context, dir string) error {
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
					if _, found := chars[chNum]; found {
						return fmt.Errorf("duplicate character %d", chNum)
					}
					x0 := bounds.Min.X + ii*charWidth
					y0 := bounds.Min.Y + jj*charHeight
					if debugFlag {
						log.Printf("importing char %d from image %v @%d,%d", chNum, name, x0, y0)
					}
					if err := builder.SetImage(im, x0, y0); err != nil {
						return err
					}
					chars[chNum] = builder.Char()
				}
			}
		}
	}
	return buildMCM(ctx, chars)
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func buildFromPNGAction(ctx *cli.Context, filename string) error {
	cols := ctx.Int("columns")
	margin := ctx.Int("margin")
	rows := int(math.Ceil(float64(mcmCharNum) / float64(cols)))
	imageWidth := (charWidth+margin)*cols + margin
	imageHeight := (charHeight+margin)*rows + margin

	chars := make(map[int]*MCMChar)
	var builder charBuilder

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	img, imfmt, err := image.Decode(f)
	if err != nil {
		return err
	}
	if imfmt != "png" {
		return fmt.Errorf("%s: invalid image format %s, must be png", filename, imfmt)
	}

	bounds := img.Bounds()
	if bounds.Dx() != imageWidth {
		return fmt.Errorf("invalid image width %d, must be %d", bounds.Dx(), imageWidth)
	}
	if bounds.Dy() != imageHeight {
		return fmt.Errorf("invalid image height %d, must be %d", bounds.Dy(), imageHeight)
	}

	for ii := 0; ii < cols; ii++ {
		for jj := 0; jj < rows; jj++ {
			leftX := ii*(charWidth+margin) + margin
			rightX := leftX + charWidth + margin
			topY := jj*(charHeight+margin) + margin
			bottomY := topY + charHeight + margin

			chNum := jj*cols + ii

			if debugFlag {
				log.Printf("importing char %d from image %v @%d,%d", chNum, filename, leftX, topY)
			}

			r := image.Rect(leftX, topY, rightX, bottomY)
			sub := img.(subImager).SubImage(r)
			if err := builder.SetImage(sub, 0, 0); err != nil {
				return err
			}
			chars[chNum] = builder.Char()
		}
	}
	return buildMCM(ctx, chars)
}

func buildAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("build requires 2 arguments, see help build")
	}
	input := ctx.Args().Get(0)
	st, err := os.Stat(input)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return buildFromDirAction(ctx, input)
	}
	return buildFromPNGAction(ctx, input)
}
