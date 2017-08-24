package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	forceFlag = false
)

func openOutputFile(filename string) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if !forceFlag {
		flags |= os.O_EXCL
	}
	f, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		if os.IsExist(err) && !forceFlag {
			for {
				fmt.Printf("File %v already exists, would you like to overwrite it? [y/N/a]: ", filename)
				r := bufio.NewReader(os.Stdin)
				line, _ := r.ReadString('\n')
				switch strings.ToLower(strings.TrimSpace(line)) {
				case "a":
					forceFlag = true
					fallthrough
				case "y":
					flags &= ^os.O_EXCL
					return os.OpenFile(filename, flags, 0644)
				case "n", "":
					return nil, err
				}
			}
		}
		return nil, err
	}
	return f, nil
}
