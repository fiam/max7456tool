package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"strconv"
	"strings"

	"github.com/fiam/max7456tool/mcm"
	"gopkg.in/yaml.v2"
)

func toInt64(i interface{}) (int64, error) {
	switch x := i.(type) {
	case int:
		return int64(x), nil
	case string:
		if len(x) == 1 {
			return int64(x[0]), nil
		}
		base := 10
		if strings.HasPrefix(strings.ToLower(x), "0x") {
			base = 16
			x = x[2:]
		}
		return strconv.ParseInt(x, base, 64)
	default:
		return 0, fmt.Errorf("can't convert %T to int64", i)
	}
}

type charBinaryData struct {
	Data     []byte
	Metadata []byte
}

func (c *charBinaryData) MergeTo(chr *mcm.Char) (*mcm.Char, error) {
	if len(c.Data) > 0 {
		return nil, errors.New("character has both visible extra data")
	}
	data := chr.Data()
	var buf bytes.Buffer
	if _, err := buf.Write(data[:mcm.MinCharBytes]); err != nil {
		return nil, err
	}
	for buf.Len() < mcm.MinCharBytes {
		if err := buf.WriteByte(mcmTransparentByte); err != nil {
			return nil, err
		}
	}
	if _, err := buf.Write(c.Metadata); err != nil {
		return nil, err
	}
	for buf.Len() < mcm.CharBytes {
		if err := buf.WriteByte(mcmTransparentByte); err != nil {
			return nil, err
		}
	}
	return mcm.NewCharFromData(buf.Bytes())
}

func (c *charBinaryData) Char() (*mcm.Char, error) {
	total := len(c.Data) + len(c.Metadata)
	if total > mcm.CharBytes {
		return nil, fmt.Errorf("character has too many bytes (%d+%d)=%d > %d",
			len(c.Data), len(c.Metadata), total, mcm.CharBytes)
	}
	if total == 0 {
		return nil, errors.New("character is empty")
	}
	maxMetadata := mcm.CharBytes - mcm.MinCharBytes
	if len(c.Metadata) > maxMetadata {
		return nil, fmt.Errorf("character metadata with %d bytes exceeds the maximum %d",
			len(c.Metadata), maxMetadata)
	}
	var buf bytes.Buffer
	if len(c.Data) > 0 {
		if _, err := buf.Write(c.Data); err != nil {
			return nil, err
		}
	}
	if len(c.Metadata) > 0 {
		// Metadata goes into the last 10 bytes
		for buf.Len() < mcm.MinCharBytes {
			if err := buf.WriteByte(mcmTransparentByte); err != nil {
				return nil, err
			}
		}
		if _, err := buf.Write(c.Metadata); err != nil {
			return nil, err
		}
	}
	for buf.Len() < mcm.CharBytes {
		if err := buf.WriteByte(mcmTransparentByte); err != nil {
			return nil, err
		}
	}
	return mcm.NewCharFromData(buf.Bytes())
}

func (c *charBinaryData) addValues(m map[interface{}]interface{}, key string, data *[]byte) error {
	val := m[key]
	if val != nil {
		slice, ok := val.([]interface{})
		if !ok {
			return fmt.Errorf("%s key is not a slice, it's %T", key, val)
		}
		var buf bytes.Buffer
		for ii, v := range slice {
			vm, ok := v.(map[interface{}]interface{})
			if !ok {
				return fmt.Errorf("key %s, entry %d is not a map, it's %T", key, ii, v)
			}
			if len(vm) != 1 {
				return fmt.Errorf("map in key %s at entry %d contains %d keys", key, ii+1, len(vm))
			}
			for kk, vv := range vm {
				ks, ok := kk.(string)
				if !ok {
					return fmt.Errorf("key %v inside %s is not string, it's %T", kk, key, kk)
				}
				switch ks {
				case "s":
					vs, ok := vv.(string)
					if !ok {
						return fmt.Errorf("argument to s must be a string, it's %v (%T)", vv, vv)
					}
					if _, err := buf.Write([]byte(vs)); err != nil {
						return err
					}
				case "u8":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i > math.MaxUint8 {
						return fmt.Errorf("can't encode %v as uint8", i)
					}
					if err := buf.WriteByte(byte(i)); err != nil {
						return err
					}
				case "i8":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i < math.MinInt8 || i > math.MaxInt8 {
						return fmt.Errorf("can't encode %v as int8", i)
					}
					if err := buf.WriteByte(byte(i)); err != nil {
						return err
					}
				case "lu16":
					fallthrough
				case "bu16":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i > math.MaxUint16 {
						return fmt.Errorf("can't encode %v as uint16", i)
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bu16" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, uint16(i)); err != nil {
						return err
					}
				case "li16":
					fallthrough
				case "bi16":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i < math.MinInt16 || i > math.MaxInt16 {
						return fmt.Errorf("can't encode %v as int16", i)
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bi16" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, int16(i)); err != nil {
						return err
					}
				case "lu32":
					fallthrough
				case "bu32":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i > math.MaxUint32 {
						return fmt.Errorf("can't encode %v as uint32", i)
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bu32" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, uint32(i)); err != nil {
						return err
					}
				case "li32":
					fallthrough
				case "bi32":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i < math.MinInt32 || i > math.MaxInt32 {
						return fmt.Errorf("can't encode %v as int32", i)
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bi32" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, int32(i)); err != nil {
						return err
					}
				case "lu64":
					fallthrough
				case "bu64":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bu64" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, uint64(i)); err != nil {
						return err
					}
				case "li64":
					fallthrough
				case "bi64":
					i, err := toInt64(vv)
					if err != nil {
						return err
					}
					if i < math.MinInt64 || i > math.MaxInt64 {
						return fmt.Errorf("can't encode %v as iint32", i)
					}
					bo := binary.ByteOrder(binary.LittleEndian)
					if ks == "bi64" {
						bo = binary.BigEndian
					}
					if err := binary.Write(&buf, bo, int64(i)); err != nil {
						return err
					}
				default:
					return fmt.Errorf("can't encode value with key %q within %s", ks, key)
				}
			}
		}
		*data = append(*data, buf.Bytes()...)
	}
	return nil
}

func (c *charBinaryData) Add(d interface{}) error {
	m, ok := d.(map[interface{}]interface{})
	if !ok {
		return fmt.Errorf("can't add data from %T", d)
	}
	if err := c.addValues(m, "data", &c.Data); err != nil {
		return err
	}
	if err := c.addValues(m, "metadata", &c.Metadata); err != nil {
		return err
	}
	return nil
}

type fontDataSet struct {
	dataSet map[int]*charBinaryData
}

func newFontDataSet() *fontDataSet {
	return &fontDataSet{
		dataSet: make(map[int]*charBinaryData),
	}
}

func (fs *fontDataSet) Values() map[int]*charBinaryData {
	return fs.dataSet
}

func (fs *fontDataSet) ParseFile(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var m map[int]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return err
	}
	for k, v := range m {
		chr := fs.dataSet[k]
		if chr == nil {
			chr = &charBinaryData{}
			fs.dataSet[k] = chr
		}
		if err := chr.Add(v); err != nil {
			return fmt.Errorf("error parsing extra data from %s: %v", filename, err)
		}
	}
	return nil
}

/*
func buildMetadata(metadata string) (*fontMetadata, error) {
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
			default:
				return nil, fmt.Errorf("unknown endianess %q", string(vtyp[0]))
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
			default:
				return nil, fmt.Errorf("unknown metadata type %q", vtyp[1:])
			}
			if err := binary.Write(&buf, byteOrder, ev); err != nil {
				return nil, err
			}
		}
		for buf.Len() < mcm.CharBytes {
			buf.WriteByte(mcmTransparentByte)
		}
		mcmCh, err := mcm.NewCharFromData(buf.Bytes())
		if err != nil {
			return nil, err
		}
		meta.data[ch] = mcmCh
	}
	return meta, nil
}
*/
