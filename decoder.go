package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	max7456Hdr = "MAX7456\r\n"
)

// MCMDecoder decodes a .mcm file into its characters.
// Use NewDecoder to initialize a decoder.
type MCMDecoder struct {
	chars []*MCMChar
}

// NChars returns the number of characters found in the
// character map. Must always be 256 for standard
// character maps.
func (d *MCMDecoder) NChars() int {
	return len(d.chars)
}

// CharAt returns the character at the given index.
func (d *MCMDecoder) CharAt(i int) *MCMChar {
	return d.chars[i]
}

// NewDecoder initializes an MCMDecoder reading the
// data from the given reader. The data must represent
// a well formed MAX7456 character map. Otherwise, this
// function will return an error.
func NewDecoder(r io.Reader) (*MCMDecoder, error) {
	br := bufio.NewReader(r)
	hdr, err := br.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if hdr != max7456Hdr {
		return nil, fmt.Errorf("unknown character map header %q", hdr)
	}
	var builder charBuilder
	builder.Reset()
	var chars []*MCMChar
	appendPixel := func(s string) error {
		val, err := strconv.ParseUint(s, 2, 32)
		if err != nil {
			return err
		}
		return builder.AppendPixel(MCMPixel(val))
	}
	ii := 2
	for {
		line, err := br.ReadString('\n')
		if err != nil && line == "" {
			if err == io.EOF && builder.IsEmpty() {
				break
			}
			return nil, err
		}
		line = strings.TrimSpace(line)
		if len(line) != 8 {
			return nil, fmt.Errorf("line %d has invalid length %d (must be 8)", ii, len(line))
		}
		ii++
		if err := appendPixel(line[0:2]); err != nil {
			return nil, err
		}
		if err := appendPixel(line[2:4]); err != nil {
			return nil, err
		}
		if err := appendPixel(line[4:6]); err != nil {
			return nil, err
		}
		if err := appendPixel(line[6:8]); err != nil {
			return nil, err
		}
		if builder.IsComplete() {
			chars = append(chars, builder.Char())
			builder.Reset()
		}
	}
	return &MCMDecoder{
		chars: chars,
	}, nil
}
