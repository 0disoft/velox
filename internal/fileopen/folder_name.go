package fileopen

import "unicode/utf8"

const MaxFolderTextNameUTF16 = 255

func ValidateFolderTextName(name string) error {
	if err := validateBaseName(name, utf8.UTFMax*MaxFolderTextNameUTF16); err != nil {
		return err
	}
	units := 0
	for _, char := range name {
		units++
		if char > 0xffff {
			units++
		}
		if units > MaxFolderTextNameUTF16 {
			return ErrUnsupported
		}
	}
	return nil
}
