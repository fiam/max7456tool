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

type charMap map[int]*mcm.Char

type namedFont struct {
	Name  string
	Chars charMap
}

type buildOptions struct {
	NoBlanks         bool
	Margin           int
	Columns          int
	RemoveDuplicates bool
}

func newBuildOptions(ctx *cli.Context) (*buildOptions, error) {
	return &buildOptions{
		NoBlanks:         ctx.Bool("no-blanks"),
		Margin:           ctx.Int("margin"),
		Columns:          ctx.Int("columns"),
		RemoveDuplicates: ctx.Bool("remove-duplicates"),
	}, nil
}

func buildMCM(output string, enc *mcm.Encoder) error {
	f, err := openOutputFile(output)
	if err != nil {
		return err
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

func loadFontFromDir(dir string) (charMap, error) {
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
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
				return nil, err
			}
			im, imfmt, err := image.Decode(imf)
			if err != nil {
				return nil, fmt.Errorf("error decoding %s: %v", filename, err)
			}
			if err := imf.Close(); err != nil {
				return nil, err
			}
			if imfmt != "png" {
				return nil, fmt.Errorf("%s: invalid image format %s, must be png", filename, imfmt)
			}
			nonExt := name[:len(name)-len(ext)]
			// Parse the name. It might contain multiple characters
			nums, err := parseFilenameCharacterNums(nonExt, im)
			if err != nil {
				return nil, err
			}
			bounds := im.Bounds()
			xw := bounds.Dx() / mcm.CharWidth
			// Import each character
			for ii, chNum := range nums {
				if _, found := chars[chNum]; found {
					return nil, fmt.Errorf("duplicate character %d", chNum)
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
					return nil, err
				}
				chars[chNum] = mcmCh

			}
		}
	}
	return chars, nil
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func loadFontFromPNG(filename string, opts *buildOptions) (charMap, error) {
	cols := opts.Columns
	margin := opts.Margin
	rows := int(math.Ceil(float64(mcm.CharNum) / float64(cols)))
	extendedRows := int(math.Ceil(float64(mcm.ExtendedCharNum) / float64(cols)))
	imageWidth := (mcm.CharWidth+margin)*cols + margin
	imageHeight := (mcm.CharHeight+margin)*rows + margin
	extendedImageHeight := (mcm.CharHeight+margin)*extendedRows + margin

	chars := make(map[int]*mcm.Char)

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, imfmt, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	if imfmt != "png" {
		return nil, fmt.Errorf("%s: invalid image format %s, must be png", filename, imfmt)
	}

	bounds := img.Bounds()
	if bounds.Dx() != imageWidth {
		return nil, fmt.Errorf("invalid image width %d, must be %d", bounds.Dx(), imageWidth)
	}
	if bounds.Dy() != imageHeight {
		if bounds.Dy() != extendedImageHeight {
			return nil, fmt.Errorf("invalid image height %d, must be %d (%d characters) or %d (%d characters)",
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
				return nil, err
			}
			if !chr.IsBlank() {
				chars[chNum] = chr
			}
		}
	}
	return chars, nil
}

func loadFontFromInput(input string, opts *buildOptions) (charMap, error) {
	st, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	var chars charMap
	if st.IsDir() {
		chars, err = loadFontFromDir(input)
	} else {
		chars, err = loadFontFromPNG(input, opts)
	}
	if err != nil {
		return nil, err
	}
	return chars, nil
}

func charIsEqualEnough(src, dst *mcm.Char) bool {
	if src.Equal(dst) {
		return true
	}
	if src.VisibleEqual(dst) && dst.MetadataIsBlank() {
		return true
	}
	return false
}

func buildFromInput(output string, input string, fontData *fontDataSet, parents []*namedFont, opts *buildOptions) (charMap, error) {
	chars, err := loadFontFromInput(input, opts)
	if err != nil {
		return nil, err
	}
	enc := &mcm.Encoder{
		Chars: chars,
		Fill:  !opts.NoBlanks,
	}

	// Fill characters from parents (if any)

	// Note that the child font might have only characters < 256, but the parent
	// fonts might have a second page
	charNum := enc.CharNum()
	for _, p := range parents {
		penc := &mcm.Encoder{Chars: p.Chars}
		if penc.CharNum() > charNum {
			charNum = penc.CharNum()
		}
	}
	for ii := 0; ii < charNum; ii++ {
		chr := chars[ii]
		if chr != nil {
			// Log a verbose message if this character is duplicated from any
			for _, p := range parents {
				if pchr := p.Chars[ii]; pchr != nil {
					if charIsEqualEnough(pchr, chr) {
						if opts.RemoveDuplicates {
							// We can only remove duplicates from the child if it's
							// a directory, otherwise it gets too messy
							if filepath.Ext(input) == "" {
								filename := filepath.Join(input, fmt.Sprintf("%03d.png", ii))
								if err := os.Remove(filename); err != nil {
									logVerbose("could not remove duplicate character %03d in %s: %v", ii, input, err)
								} else {
									logVerbose("removed duplicate character %03d in %s, since it's equal to its parent %s",
										ii, input, p.Name)
								}

							} else {
								logVerbose("not removing duplicate character %03d in %s because the source is an image - switch to a directory based format to use this option",
									ii, input)
							}
						} else {
							logVerbose("character %03d in %s is equal to parent font %s and can be removed",
								ii, input, p.Name)
						}
					}
				}
			}
		} else {
			// Check if we can fill it from the parents
			for _, p := range parents {
				if pchr := p.Chars[ii]; pchr != nil {
					logDebug("filling character %03d in %s from parent font %s", ii, output, p.Name)
					// Check if we have different metadata for this character in the child font.
					// In that case, we overwrite it.
					charData := fontData.Values()[ii]
					if charData != nil {
					}
					chars[ii] = pchr
					break
				}
			}
		}
	}

	// Apply extra font data
	if fontData != nil {
		for k, v := range fontData.Values() {
			if prev, found := chars[k]; found {
				repl, err := v.MergeTo(k, prev)
				if err != nil {
					return nil, fmt.Errorf("error merging binary data into existing character %d: %v", k, err)
				}
				chars[k] = repl
			} else {
				chr, err := v.Char()
				if err != nil {
					return nil, fmt.Errorf("error decoding binary character %d: %v", k, err)
				}
				logVerbose("creating new character %03d from extra data in font %s", k, input)
				chars[k] = chr
			}
		}
	}

	if err := buildMCM(output, enc); err != nil {
		return nil, err
	}
	return chars, nil
}

func buildAction(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return errors.New("build requires 2 arguments, see help build")
	}
	opts, err := newBuildOptions(ctx)
	if err != nil {
		return err
	}
	input := ctx.Args().Get(0)
	output := ctx.Args().Get(1)
	fontData := newFontDataSet()
	for _, e := range ctx.StringSlice("extra") {
		if err := fontData.ParseFile(e); err != nil {
			return err
		}
	}
	_, err = buildFromInput(output, input, fontData, nil, opts)
	return err
}
