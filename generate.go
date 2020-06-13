package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v2"
)

type generateFontConfig struct {
	Source    string      `yaml:"source"`
	ExtraData interface{} `yaml:"extra"`
	Output    string      `yaml:"output"`
}

func (c *generateFontConfig) ExtraDataFiles(dir string) ([]string, error) {
	var files []string
	switch x := c.ExtraData.(type) {
	case []interface{}:
		for ii, v := range x {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("item %d in extra data for source %s is not a string, it's %T", ii+1, c.Source, v)
			}
			if s != "" {
				files = append(files, filepath.Join(dir, s))
			}
		}
	case string:
		if x != "" {
			files = append(files, filepath.Join(dir, x))
		}
	case bool:
		if x {
			ext := filepath.Ext(c.Source)
			nonExt := c.Source[:len(c.Source)-len(ext)]
			files = append(files, filepath.Join(dir, nonExt+".yaml"))
		}
	default:
		return nil, fmt.Errorf("can't specify extra data files as %T = %v", x, x)
	}
	return files, nil
}

type generateConfig struct {
	Previews    bool                  `yaml:"previews"`
	ExtraData   []string              `yaml:"extra"`
	DefaultFont string                `yaml:"default"`
	Fonts       []*generateFontConfig `yaml:"fonts"`
	Dir         string                `yaml:"-"`
}

func (c *generateConfig) ExtraDataFiles() []string {
	files := make([]string, len(c.ExtraData))
	for ii, v := range c.ExtraData {
		files[ii] = filepath.Join(c.Dir, v)
	}
	return files
}

func (c *generateConfig) defaultFont() (*generateFontConfig, error) {
	if c.DefaultFont != "" {
		for _, v := range c.Fonts {
			if c.DefaultFont == v.Source {
				return v, nil
			}
		}
		// This should not happen due to validate(), but better
		// be safe than sorry
		return nil, fmt.Errorf("could not find default font %q", c.DefaultFont)
	}
	return nil, nil
}

func (c *generateConfig) Parents(cfg *generateFontConfig) ([]*generateFontConfig, error) {
	var parents []*generateFontConfig
	// The only parent font we support for now is the default font
	def, err := c.defaultFont()
	if err != nil {
		return nil, err
	}
	if def != nil && def.Source != cfg.Source {
		parents = append(parents, def)
	}
	return parents, nil
}

func (c *generateConfig) Load(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("error reading config file %s: %v", filename, err)
	}
	if err := yaml.Unmarshal(data, c); err != nil {
		return err
	}
	// Store filename's directory for relative paths
	c.Dir = filepath.Dir(filename)
	return c.validate()
}

func (c *generateConfig) validate() error {
	// Ensure all ExtraData files exist
	for _, v := range c.ExtraDataFiles() {
		st, err := os.Stat(v)
		if err != nil {
			return fmt.Errorf("global extra data file %s is not readable: %v", v, err)
		}
		if st.IsDir() {
			return fmt.Errorf("global extra data file %s is a directory, not a file", v)
		}
	}
	// If default is non-empty, ensure it exists
	if c.DefaultFont != "" {
		found := false
		var names []string // collect this for the potential error message
		for _, v := range c.Fonts {
			names = append(names, strconv.Quote(v.Source))
			if v.Source == c.DefaultFont {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("default font %q not found in the fonts list (%s)", c.DefaultFont, strings.Join(names, ", "))
		}
	}
	// Ensure all input sources exist
	for ii, v := range c.Fonts {
		if v.Source == "" {
			return fmt.Errorf("source %d is empty", ii+1)
		}
		p := filepath.Join(c.Dir, v.Source)
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("source %q (%q) doesn't exist: %v", v.Source, p, err)
		}
	}
	return nil
}

func generateFont(ctx *cli.Context, globalFontData *fontDataSet, config *generateConfig,
	font *generateFontConfig, opts *buildOptions, charMaps map[string]charMap) (charMap, error) {

	// Check if this font was already built due to
	// being the parent of a previous font
	if m := charMaps[font.Source]; m != nil {
		return m, nil
	}
	var parentFonts []*namedFont
	parents, err := config.Parents(font)
	if err != nil {
		return nil, err
	}
	for _, p := range parents {
		pmcm := charMaps[p.Source]
		if pmcm == nil {
			// Must generate the parent first
			if _, err := generateFont(ctx, globalFontData, config, p, opts, charMaps); err != nil {
				return nil, err
			}
			pmcm = charMaps[p.Source]
			if pmcm == nil {
				panic(fmt.Errorf("generating font for %q didn't fill its charMap entry", p.Source))
			}
		}
		parentFonts = append(parentFonts, &namedFont{Name: p.Source, Chars: pmcm})
	}

	logVerbose("generating font from %q", font.Source)
	p := filepath.Join(config.Dir, font.Source)
	ext := filepath.Ext(p)
	nonExt := p[:len(p)-len(ext)]
	fontData := globalFontData.Clone()
	extraDataFiles, err := font.ExtraDataFiles(config.Dir)
	if err != nil {
		return nil, err
	}
	for _, f := range extraDataFiles {
		logVerbose("parsing extra data from %q", f)
		if err := fontData.ParseFile(f); err != nil {
			return nil, err
		}
	}
	var output string
	if font.Output != "" {
		output = filepath.Join(config.Dir, font.Output)
	} else {
		output = nonExt + ".mcm"
	}
	logVerbose("generating font %q from %q", output, p)
	charMap, err := buildFromInput(output, p, fontData, parentFonts, opts)
	if err != nil {
		return nil, err
	}
	if config.Previews {
		pngOutput := nonExt + ".png"
		logVerbose("generating preview image %q from %q", pngOutput, output)
		if err := buildPNGFromMCM(ctx, pngOutput, output); err != nil {
			return nil, err
		}
	}
	charMaps[font.Source] = charMap
	return charMap, nil
}

func generateAction(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("generate requires 1 argument, see help generate")
	}
	opts, err := newBuildOptions(ctx)
	if err != nil {
		return err
	}
	configFile := ctx.Args().Get(0)
	var config generateConfig
	if err := config.Load(configFile); err != nil {
		return err
	}
	globalFontData := newFontDataSet()
	for _, c := range config.ExtraDataFiles() {
		logVerbose("parsing global extra data from %q", c)
		if err := globalFontData.ParseFile(c); err != nil {
			return err
		}
	}
	charMaps := make(map[string]charMap)
	for _, v := range config.Fonts {
		mcm, err := generateFont(ctx, globalFontData, &config, v, opts, charMaps)
		if err != nil {
			return err
		}
		charMaps[v.Source] = mcm
	}
	return nil
}
