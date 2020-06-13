package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

const (
	defaultMargin  = 1
	defaultColumns = 16
)

const (
	extraDataUsage = `Extra data to add to the font. Either as a yaml file or as a string.
	This can be used to to store metadata in the non visible region of a character
	or to define entire characters used for data storage.
	For an example file, see https://github.com/fiam/max7456tool/example_data.yaml

	Multiple files can be combined by repeating this flag.`
)

var (
	verboseFlag = false
	debugFlag   = false
	appVersion  string
)

func main() {
	app := cli.NewApp()
	app.Version = appVersion
	buildAndGenerateFlags := []cli.Flag{
		&cli.BoolFlag{
			Name:    "no-blanks",
			Aliases: []string{"nb"},
			Usage:   "Don't fill missing characters with blanks (used only for directory input)",
		},
		&cli.IntFlag{
			Name:    "margin",
			Aliases: []string{"m"},
			Value:   defaultMargin,
			Usage:   "Margin between each character (used only for image input)",
		},
		&cli.IntFlag{
			Name:    "columns",
			Aliases: []string{"c"},
			Value:   defaultColumns,
			Usage:   "Number of columns in the output image (used only for image input)",
		},
	}
	var buildFlags []cli.Flag
	buildFlags = append(buildFlags, buildAndGenerateFlags...)
	buildFlags = append(buildFlags, &cli.StringSliceFlag{
		Name:    "extra",
		Aliases: []string{"e"},
		Usage:   extraDataUsage,
	})
	var generateFlags []cli.Flag
	generateFlags = append(generateFlags, buildAndGenerateFlags...)
	generateFlags = append(generateFlags, &cli.BoolFlag{
		Name:  "remove-duplicates",
		Usage: "Remove duplicate characters that are the same in the child and parent font",
	})
	app.Usage = "tool for managing .mcm character sets for MAX7456"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:        "force",
			Aliases:     []string{"f"},
			Usage:       "Overwrite output files without asking",
			Destination: &forceFlag,
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Aliases:     []string{"v"},
			Usage:       "Enable verbose output",
			Destination: &verboseFlag,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Usage:       "Print debug messages",
			Destination: &debugFlag,
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:      "extract",
			Usage:     "Extract all characters to individual images",
			ArgsUsage: "<input.mcm> <output-dir>",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "add-blanks",
					Aliases: []string{"b"},
					Usage:   "Include blank characters in the extracted files",
				},
			},
			Action: extractAction,
		},
		{
			Name:      "build",
			Usage:     "Build a .mcm from the files in the given directory or .png file",
			ArgsUsage: "<input> <output.mcm>",
			Flags:     buildFlags,
			Action:    buildAction,
		},
		{
			Name:      "generate",
			Usage:     "Generate multiple fonts from a fonts.yaml file",
			ArgsUsage: "<fonts.yaml> # See an example at https://github.com/fiam/max7456tool/example_fonts.yaml",
			Flags:     generateFlags,
			Action:    generateAction,
		},
		{
			Name:      "png",
			Usage:     "Generate a .png from an .mcm",
			ArgsUsage: "<input.mcm> <output.png>",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "margin",
					Aliases: []string{"m"},
					Value:   defaultMargin,
					Usage:   "Margin between each character",
				},
				&cli.IntFlag{
					Name:    "columns",
					Aliases: []string{"c"},
					Value:   defaultColumns,
					Usage:   "Number of columns in the output image",
				},
			},
			Action: pngAction,
		},
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
