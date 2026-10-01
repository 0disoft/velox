package pebranding

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/png"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const MaxIconBytes = 2 << 20

type Options struct {
	Enabled                                        bool
	Name, Version, Company, Description, Copyright string
	Icon                                           []byte
}

type resource struct {
	kind, id uint16
	data     []byte
}

func Validate(options Options) error {
	if !options.Enabled {
		return nil
	}
	for _, value := range []string{options.Name, options.Version, options.Company, options.Description, options.Copyright} {
		if !utf8.ValidString(value) || len(utf16.Encode([]rune(value))) > 256 || strings.ContainsFunc(value, unicode.IsControl) {
			return errors.New("branding text must be valid text of at most 256 UTF-16 units without control characters")
		}
	}
	if _, err := numericVersion(options.Version); err != nil {
		return err
	}
	if len(options.Icon) > 0 {
		_, err := iconResources(options.Icon)
		return err
	}
	return nil
}

func numericVersion(value string) ([4]uint16, error) {
	var result [4]uint16
	base := strings.SplitN(strings.SplitN(value, "+", 2)[0], "-", 2)[0]
	parts := strings.Split(base, ".")
	if len(parts) < 1 || len(parts) > 4 {
		return result, errors.New("branding requires a numeric app.version with one to four components")
	}
	for i, part := range parts {
		if part == "" || strings.ContainsFunc(part, func(r rune) bool { return r < '0' || r > '9' }) {
			return result, errors.New("branding requires a numeric app.version")
		}
		value, err := strconv.ParseUint(part, 10, 16)
		if err != nil {
			return result, fmt.Errorf("branding version component: %w", err)
		}
		result[i] = uint16(value)
	}
	return result, nil
}

func iconResources(data []byte) ([]resource, error) {
	if len(data) < 6 || len(data) > MaxIconBytes || binary.LittleEndian.Uint16(data) != 0 || binary.LittleEndian.Uint16(data[2:]) != 1 {
		return nil, errors.New("branding icon must be an ICO file of at most 2 MiB")
	}
	count := int(binary.LittleEndian.Uint16(data[4:]))
	if count < 1 || count > 32 || len(data) < 6+16*count {
		return nil, errors.New("branding icon has an invalid image count")
	}
	group := append([]byte(nil), data[:6]...)
	resources := make([]resource, 0, count+2)
	for i := 0; i < count; i++ {
		entry := data[6+16*i : 6+16*(i+1)]
		size, offset := uint64(binary.LittleEndian.Uint32(entry[8:])), uint64(binary.LittleEndian.Uint32(entry[12:]))
		if size == 0 || offset < uint64(6+16*count) || offset+size > uint64(len(data)) || entry[3] != 0 {
			return nil, errors.New("branding icon image lies outside its file")
		}
		image := data[offset : offset+size]
		width, height := int(entry[0]), int(entry[1])
		if width == 0 {
			width = 256
		}
		if height == 0 {
			height = 256
		}
		if bytes.HasPrefix(image, []byte("\x89PNG\r\n\x1a\n")) {
			config, err := png.DecodeConfig(bytes.NewReader(image))
			if err != nil || config.Width != width || config.Height != height {
				return nil, errors.New("branding icon PNG dimensions do not match its directory")
			}
		} else {
			if len(image) < 40 || binary.LittleEndian.Uint32(image) != 40 || int32(binary.LittleEndian.Uint32(image[4:])) != int32(width) || int32(binary.LittleEndian.Uint32(image[8:])) != int32(height*2) {
				return nil, errors.New("branding icon requires PNG or BITMAPINFOHEADER images with matching dimensions")
			}
			bits := int(binary.LittleEndian.Uint16(image[14:]))
			if binary.LittleEndian.Uint16(image[12:]) != 1 || binary.LittleEndian.Uint32(image[16:]) != 0 || (bits != 1 && bits != 4 && bits != 8 && bits != 16 && bits != 24 && bits != 32) {
				return nil, errors.New("branding icon bitmap format is unsupported")
			}
			palette := int(binary.LittleEndian.Uint32(image[32:]))
			if palette == 0 && bits <= 8 {
				palette = 1 << bits
			}
			if palette > 256 || len(image) < 40+palette*4+((width*bits+31)/32)*4*height {
				return nil, errors.New("branding icon bitmap pixels are truncated")
			}
		}
		id := uint16(101 + i)
		resources = append(resources, resource{3, id, append([]byte(nil), image...)})
		group = append(group, entry[:12]...)
		group = binary.LittleEndian.AppendUint16(group, id)
	}
	resources = append(resources, resource{14, 1, group}, resource{14, 11, group})
	return resources, nil
}

func versionResource(options Options, filename string) []byte {
	v, _ := numericVersion(options.Version)
	var fixed []byte
	for _, value := range []uint32{0xfeef04bd, 0x10000, uint32(v[0])<<16 | uint32(v[1]), uint32(v[2])<<16 | uint32(v[3]), uint32(v[0])<<16 | uint32(v[1]), uint32(v[2])<<16 | uint32(v[3]), 0x3f, 0, 0x40004, 1, 0, 0, 0} {
		fixed = binary.LittleEndian.AppendUint32(fixed, value)
	}
	description := options.Description
	if description == "" {
		description = options.Name
	}
	var entries [][]byte
	for _, item := range [][2]string{{"CompanyName", options.Company}, {"FileDescription", description}, {"FileVersion", options.Version}, {"InternalName", filename}, {"LegalCopyright", options.Copyright}, {"OriginalFilename", filename}, {"ProductName", options.Name}, {"ProductVersion", options.Version}} {
		if item[1] != "" {
			value := utf16Bytes(item[1])
			entries = append(entries, versionBlock(item[0], 1, uint16(len(value)/2), value))
		}
	}
	strings := versionBlock("StringFileInfo", 1, 0, nil, versionBlock("040904B0", 1, 0, nil, entries...))
	translation := versionBlock("VarFileInfo", 1, 0, nil, versionBlock("Translation", 0, 4, []byte{9, 4, 0xb0, 4}))
	return versionBlock("VS_VERSION_INFO", 0, uint16(len(fixed)), fixed, strings, translation)
}

func utf16Bytes(value string) []byte {
	var data []byte
	for _, unit := range append(utf16.Encode([]rune(value)), 0) {
		data = binary.LittleEndian.AppendUint16(data, unit)
	}
	return data
}

func versionBlock(key string, kind, valueLength uint16, value []byte, children ...[]byte) []byte {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data[2:], valueLength)
	binary.LittleEndian.PutUint16(data[4:], kind)
	data = append(data, utf16Bytes(key)...)
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	data = append(data, value...)
	for _, child := range children {
		for len(data)%4 != 0 {
			data = append(data, 0)
		}
		data = append(data, child...)
	}
	binary.LittleEndian.PutUint16(data, uint16(len(data)))
	return data
}
