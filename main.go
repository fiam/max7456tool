package main

import (
	"os"

	"github.com/urfave/cli"
)

const (
	defaultMargin  = 1
	defaultColumns = 16
)

var (
	debugFlag = false
)

func main() {
	app := cli.NewApp()
	app.Version = "0.4"
	app.Usage = "tool for managing .mcm character sets for MAX7456"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "force, f",
			Usage:       "Overwrite output files without asking",
			Destination: &forceFlag,
		},
		cli.BoolFlag{
			Name:        "debug, d",
			Usage:       "Print debug messages",
			Destination: &debugFlag,
		},
	}
	app.Commands = []cli.Command{
		{
			Name:      "extract",
			Usage:     "Extract all characters to individual images",
			ArgsUsage: "<input.mcm> <output-dir>",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "add-blanks, b",
					Usage: "Include blank characters in the extracted files",
				},
			},
			Action: extractAction,
		},
		{
			Name:      "build",
			Usage:     "Build a .mcm from the files in the given directory or .png file",
			ArgsUsage: "<input> <output.mcm>",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "no-blanks, b",
					Usage: "Don't fill missing characters with blanks (used only for directory input)",
				},
				cli.IntFlag{
					Name:  "margin, m",
					Value: defaultMargin,
					Usage: "Margin between each character (used only for image input)",
				},
				cli.IntFlag{
					Name:  "columns, c",
					Value: defaultColumns,
					Usage: "Number of columns in the output image (used only for image input)",
				},
				cli.StringFlag{
					Name:  "metadata, md",
					Value: "",
					Usage: "Metadata to add to the font. Metadata format is XXX=[(b|l)(i|u)(8|16|32|64):v]... where:\n" +
						"\tXXX represents a character number\n" +
						"\t(b|l)(i|u)(8|16|32|64) indicates the endianess, data type and bit size of the value\n" +
						"\tv represents the value to encode\n" +
						"\tValues encoded sequentially into the same character are separeted by ','" +
						"\tMetadata characters are separated by '-'\n" +
						"\tFor example: -md 255=lu8:3,li32:17-254=lu64:3",
				},
			},
			Action: buildAction,
		},
		{
			Name:      "png",
			Usage:     "Generate a .png from an .mcm",
			ArgsUsage: "<input.mcm> <output.png>",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "margin, m",
					Value: defaultMargin,
					Usage: "Margin between each character",
				},
				cli.IntFlag{
					Name:  "columns, c",
					Value: defaultColumns,
					Usage: "Number of columns in the output image",
				},
			},
			Action: pngAction,
		},
	}
	app.Run(os.Args)
}
