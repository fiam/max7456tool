package mcm

import (
	"image/color"
	"reflect"
)

var (
	BlackColor              = &color.RGBA{R: 0, G: 0, B: 0, A: 255}
	WhiteColor              = &color.RGBA{R: 255, G: 255, B: 255, A: 255}
	DefaultTransparentColor = &color.RGBA{R: 128, G: 128, B: 128, A: 255} // 50% gray
)

func isNilColor(c color.Color) bool {
	if c == nil {
		return true
	}
	v := reflect.ValueOf(c)
	return v.Kind() == reflect.Ptr && v.IsNil()
}
