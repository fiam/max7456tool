package main

import (
	"fmt"
	"io"
	"strconv"
)

const (
	mcmCharNum = 256
)

type Encoder struct {
	Chars map[int]*MCMChar
	Fill  bool
}

func (e *Encoder) Encode(w io.Writer) error {
	for k := range e.Chars {
		if k >= mcmCharNum {
			return fmt.Errorf("invalid character number %d, max is %d", k, mcmCharNum-1)
		}
	}
	if _, err := io.WriteString(w, max7456Hdr); err != nil {
		return err
	}
	for ii := 0; ii < mcmCharNum; ii++ {
		c := e.Chars[ii]
		if c == nil {
			if !e.Fill {
				return fmt.Errorf("missing character %d", ii)
			}
			c = blankCharacter
		}
		data := c.Data()
		if len(data) != charBytes {
			return fmt.Errorf("invalid character length %d (!= %d)", len(data), charBytes)
		}
		for jj, b := range data {
			if ii > 0 || jj > 0 {
				if _, err := io.WriteString(w, "\r\n"); err != nil {
					return err
				}
			}
			s := strconv.FormatUint(uint64(b), 2)
			for len(s) < 8 {
				s = "0" + s
			}
			if _, err := io.WriteString(w, s); err != nil {
				return err
			}
		}
	}
	return nil
}
