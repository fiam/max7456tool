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

	"github.com/fiam/max7456tool/mcm"

	"github.com/urfave/cli/v2"
)

const (
	// all pixels = 01
	mcmTransparentByte = 85
)

func buildMCM(ctx *cli.Context, chars map[int]*mcm.Char, extra *fontDataSet) error {
	if extra != nil {
		for k, v := range extra.Values() {
			if prev, found := chars[k]; found {
				repl, err := v.MergeTo(prev)
				if err != nil {
					return fmt.Errorf("error merging binary data into existing character %d: %v", k, err)
				}
				chars[k] = repl
			} else {
				chr, err := v.Char()
				if err != nil {
					return fmt.Errorf("error decoding binary character %d: %v", k, err)
				}
				chars[k] = chr
			}
		}
	}
	output := ctx.Args().Get(1)
	f, err := openOutputFile(output)
	if err != nil {
		return err
	}
	enc := &mcm.Encoder{
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

func parseFilenameCharacterNums(nonExt string, im image.Image) ([]int, error) {
	px := im.Bounds().Dx()
	if px%mcm.CharWidth != 0 {
		return nil, fmt.Errorf("invalid image width %d, must be a multiple of %d", px, mcm.CharWidth)
	}
	sx := px / mcm.CharWidth

	py := im.Bounds().Dy()
	if py%mcm.CharHeight != 0 {
		return nil, fmt.Errorf("invalid image height %d, must be a multiple of %d", py, mcm.CharHeight)
	}
	sy := py / mcm.CharHeight

	total := sx * sy

	segments := strings.Split(nonExt, "-")
	var nums []int
	for _, s := range segments {
		items := strings.Split(s, "_")
		prev := -1
		for _, v := range items {
			n, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q in image filename %q: %v", v, nonExt, err)
			}
			if prev >= 0 {
				for ii := prev + 1; ii <= n; ii++ {
					nums = append(nums, ii)
				}
			} else {
				nums = append(nums, n)
			}
			prev = n
		}
	}
	if len(nums) != total {
		return nil, fmt.Errorf("image %q with size %dx%d must contain %d characters, %d declared", nonExt, px, py, total, len(nums))
	}
	return nums, nil
}

func buildFromDirAction(ctx *cli.Context, dir string, extra *fontDataSet) error {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}
	chars := make(map[int]*mcm.Char)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := filepath.Ext(name)
		if strings.ToLower(ext) == ".png" {
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
			nonExt := name[:len(name)-len(ext)]
			// Parse the name. It might contain multiple characters
			nums, err := parseFilenameCharacterNums(nonExt, im)
			if err != nil {
				return err
			}
			bounds := im.Bounds()
			xw := bounds.Dx() / mcm.CharWidth
			// Import each character
			for ii, chNum := range nums {
				if _, found := chars[chNum]; found {
					return fmt.Errorf("duplicate character %d", chNum)
				}
				xc := ii % xw
				yc := ii / xw
				x0 := bounds.Min.X + xc*mcm.CharWidth
				y0 := bounds.Min.Y + yc*mcm.CharHeight
				if debugFlag {
					log.Printf("importing char %d from image %v @%d,%d", chNum, name, x0, y0)
				}
				mcmCh, err := mcm.NewCharFromImage(im, x0, y0)
				if err != nil {
					return err
				}
				chars[chNum] = mcmCh

			}
		}
	}
	return buildMCM(ctx, chars, extra)
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func buildFromPNGAction(ctx *cli.Context, filename string, extra *fontDataSet) error {
	cols := ctx.Int("columns")
	margin := ctx.Int("margin")
	rows := int(math.Ceil(float64(mcm.CharNum) / float64(cols)))
	extendedRows := int(math.Ceil(float64(mcm.ExtendedCharNum) / float64(cols)))
	imageWidth := (mcm.CharWidth+margin)*cols + margin
	imageHeight := (mcm.CharHeight+margin)*rows + margin
	extendedImageHeight := (mcm.CharHeight+margin)*extendedRows + margin

	chars := make(map[int]*mcm.Char)

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
		if bounds.Dy() != extendedImageHeight {
			return fmt.Errorf("invalid image height %d, must be %d (%d characters) or %d (%d characters)",
				bounds.Dy(), imageHeight, mcm.CharNum, extendedImageHeight, mcm.ExtendedCharNum)
		}
		rows = extendedRows
	}

	for ii := 0; ii < cols; ii++ {
		for jj := 0; jj < rows; jj++ {
			leftX := ii*(mcm.CharWidth+margin) + margin
			rightX := leftX + mcm.CharWidth + margin
			topY := jj*(mcm.CharHeight+margin) + margin
			bottomY := topY + mcm.CharHeight + margin

			chNum := jj*cols + ii

			if debugFlag {
				log.Printf("importing char %d from image %v @%d,%d", chNum, filename, leftX, topY)
			}

			r := image.Rect(leftX, topY, rightX, bottomY)
			sub := img.(subImager).SubImage(r)
			chr, err := mcm.NewCharFromImage(sub, 0, 0)
			if err != nil {
				return err
			}
			if !chr.IsBlank() {
				chars[chNum] = chr
			}
		}
	}
	return buildMCM(ctx, chars, extra)
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
	fontData := newFontDataSet()
	for _, e := range ctx.StringSlice("extra") {
		if err := fontData.ParseFile(e); err != nil {
			return err
		}
	}
	if st.IsDir() {
		return buildFromDirAction(ctx, input, fontData)
	}
	return buildFromPNGAction(ctx, input, fontData)
}
