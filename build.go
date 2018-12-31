package main

import (
	"bytes"
	"encoding/binary"
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

func buildMCM(ctx *cli.Context, chars map[int]*MCMChar, meta *fontMetadata) error {
	if meta != nil {
		for k, v := range meta.data {
			if _, found := chars[k]; found {
				return fmt.Errorf("character %d has both data and metadata", k)
			}
			chars[k] = v
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

func buildFromDirAction(ctx *cli.Context, dir string, meta *fontMetadata) error {
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
	return buildMCM(ctx, chars, meta)
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func buildFromPNGAction(ctx *cli.Context, filename string, meta *fontMetadata) error {
	cols := ctx.Int("columns")
	margin := ctx.Int("margin")
	rows := int(math.Ceil(float64(mcmCharNum) / float64(cols)))
	extendedRows := int(math.Ceil(float64(mcmExtendedCharNum) / float64(cols)))
	imageWidth := (charWidth+margin)*cols + margin
	imageHeight := (charHeight+margin)*rows + margin
	extendedImageHeight := (charHeight+margin)*extendedRows + margin

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
		if bounds.Dy() != extendedImageHeight {
			return fmt.Errorf("invalid image height %d, must be %d (%d characters) or %d (%d characters)",
				bounds.Dy(), imageHeight, mcmCharNum, extendedImageHeight, mcmExtendedCharNum)
		}
		rows = extendedRows
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
			chr := builder.Char()
			if !chr.isBlank() {
				chars[chNum] = chr
			}
		}
	}
	return buildMCM(ctx, chars, meta)
}

type fontMetadata struct {
	data map[int]*MCMChar
}

func buildMetadata(metadata string) (*fontMetadata, error) {
	meta := &fontMetadata{
		data: make(map[int]*MCMChar),
	}
	for _, c := range strings.Split(metadata, "-") {
		parts := strings.SplitN(c, "=", 2)
		ch, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid metadata character number %q: %v", parts[0], err)
		}
		var buf bytes.Buffer
		for _, p := range strings.Split(parts[1], ",") {
			vparts := strings.SplitN(p, ":", 2)
			vtyp := strings.ToLower(vparts[0])
			var byteOrder binary.ByteOrder
			switch vtyp[0] {
			case 'l':
				byteOrder = binary.LittleEndian
			case 'b':
				byteOrder = binary.BigEndian
			}
			v, err := strconv.ParseInt(vparts[1], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid value %q: %v", vparts[1], err)
			}
			var ev interface{}
			switch vtyp[1:] {
			case "u8":
				ev = uint8(v)
			case "i8":
				ev = int8(v)
			case "u16":
				ev = uint16(v)
			case "i16":
				ev = int16(v)
			case "u32":
				ev = uint32(v)
			case "i32":
				ev = int32(v)
			case "u64":
				ev = uint64(v)
			case "i64":
				ev = int64(v)
			}
			if err := binary.Write(&buf, byteOrder, ev); err != nil {
				return nil, err
			}
		}
		for buf.Len() < charBytes {
			buf.WriteByte(mcmTransparentByte)
		}
		meta.data[ch] = &MCMChar{
			data: buf.Bytes(),
		}
	}
	return meta, nil
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
	var metadata *fontMetadata
	if m := ctx.String("metadata"); m != "" {
		metadata, err = buildMetadata(m)
		if err != nil {
			return err
		}
	}
	if st.IsDir() {
		return buildFromDirAction(ctx, input, metadata)
	}
	return buildFromPNGAction(ctx, input, metadata)
}
