package main

import (
	"fmt"
	"log"
	"os"
)

var logger = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

func logVerbose(format string, v ...interface{}) error {
	if verboseFlag || debugFlag {
		logger.Output(2, fmt.Sprintf(format, v...))
	}
	return nil
}

func logDebug(format string, v ...interface{}) error {
	if debugFlag {
		logger.Output(2, fmt.Sprintf(format, v...))
	}
	return nil
}
