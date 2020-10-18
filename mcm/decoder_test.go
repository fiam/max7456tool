package mcm

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func testChar(t *testing.T, dec *Decoder, idx int, data string) {
	expected, err := hex.DecodeString(data)
	if err != nil {
		t.Fatal(err)
	}
	chr := dec.CharAt(idx)
	chrData := chr.Data()
	if len(expected) != len(chrData) {
		t.Fatalf("expecting char %d with %d bytes, got %d instead", idx, len(expected), len(chrData))
	}
	if !reflect.DeepEqual(expected, chrData) {
		t.Fatalf("expecting char \n%d = %s, got \n%d = %s", idx, hex.EncodeToString(expected), idx, hex.EncodeToString(chrData))
	}
}

func TestDecoder(t *testing.T) {
	const (
		expectedTotal = 512
		char150       = "55555555555555001554aa8554808554808554808554808554808554808554808554808554808554808554808554808554aa85550015720504030c6f00ff6302"
	)
	f, err := os.Open(filepath.Join("_testdata", "vision.mcm"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dec, err := NewDecoder(f)
	if err != nil {
		t.Fatal(err)
	}
	total := dec.NChars()
	if total != expectedTotal {
		t.Fatalf("expecting %d characters, got %d instead", expectedTotal, total)
	}
	testChar(t, dec, 150, char150)
}
