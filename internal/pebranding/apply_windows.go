//go:build windows

package pebranding

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel               = windows.NewLazySystemDLL("kernel32.dll")
	beginUpdate          = kernel.NewProc("BeginUpdateResourceW")
	update               = kernel.NewProc("UpdateResourceW")
	endUpdate            = kernel.NewProc("EndUpdateResourceW")
	enumLanguages        = kernel.NewProc("EnumResourceLanguagesW")
	languageResults      sync.Map
	languageSequence     atomic.Uintptr
	languageCallback     uintptr
	languageCallbackOnce sync.Once
)

func Apply(path, filename string, options Options) error {
	if !options.Enabled {
		return nil
	}
	if err := Validate(options); err != nil {
		return err
	}
	file, err := pe.Open(path)
	if err != nil {
		return err
	}
	header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	symbolOffset, symbolCount := file.PointerToSymbolTable, file.NumberOfSymbols
	file.Close()
	if !ok {
		return errors.New("branding requires a PE32+ executable")
	}
	if header.DataDirectory[4].VirtualAddress != 0 || header.DataDirectory[4].Size != 0 {
		return errors.New("branding cannot modify a signed executable; use an unsigned template and sign the final app separately")
	}
	var symbols []byte
	if symbolOffset != 0 {
		input, err := os.Open(path)
		if err != nil {
			return err
		}
		info, err := input.Stat()
		if err == nil && (int64(symbolOffset) > info.Size() || info.Size()-int64(symbolOffset) > 16<<20) {
			err = errors.New("branding symbol table exceeds its size budget")
		}
		if err == nil {
			symbols, err = io.ReadAll(io.NewSectionReader(input, int64(symbolOffset), info.Size()-int64(symbolOffset)))
		}
		input.Close()
		if err != nil {
			return err
		}
	}
	resources := []resource{{16, 1, versionResource(options, filename)}}
	if len(options.Icon) > 0 {
		icons, err := iconResources(options.Icon)
		if err != nil {
			return err
		}
		resources = append(resources, icons...)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	// Discover existing group languages before editing, then unload the data file.
	module, err := windows.LoadLibraryEx(path, 0, windows.LOAD_LIBRARY_AS_DATAFILE)
	if err != nil {
		return err
	}
	languages := map[uint16][]uint16{}
	languageCallbackOnce.Do(func() {
		languageCallback = windows.NewCallback(func(_ uintptr, _ uintptr, _ uintptr, language uintptr, token uintptr) uintptr {
			if value, ok := languageResults.Load(token); ok {
				found := value.(*[]uint16)
				*found = append(*found, uint16(language))
			}
			return 1
		})
	})
	for _, id := range []uint16{1, 11} {
		var found []uint16
		token := languageSequence.Add(1)
		languageResults.Store(token, &found)
		enumLanguages.Call(uintptr(module), 14, uintptr(id), languageCallback, token)
		languageResults.Delete(token)
		if len(found) == 0 {
			found = []uint16{0x409}
		}
		languages[id] = found
	}
	windows.FreeLibrary(module)
	handle, _, callErr := beginUpdate.Call(uintptr(unsafe.Pointer(name)), 0)
	if handle == 0 {
		return fmt.Errorf("begin branding resources: %w", callErr)
	}
	committed := false
	defer func() {
		if !committed {
			endUpdate.Call(handle, 1)
		}
	}()
	iconLanguages := map[uint16]bool{}
	for _, langs := range languages {
		for _, lang := range langs {
			iconLanguages[lang] = true
		}
	}
	for _, item := range resources {
		langs := []uint16{0x409}
		if item.kind == 14 {
			langs = languages[item.id]
		}
		if item.kind == 3 {
			langs = nil
			for lang := range iconLanguages {
				langs = append(langs, lang)
			}
		}
		slices.Sort(langs)
		for _, language := range langs {
			result, _, err := update.Call(handle, uintptr(item.kind), uintptr(item.id), uintptr(language), uintptr(unsafe.Pointer(&item.data[0])), uintptr(len(item.data)))
			runtime.KeepAlive(item.data)
			if result == 0 {
				return fmt.Errorf("write branding resource: %w", err)
			}
		}
	}
	result, _, err := endUpdate.Call(handle, 0)
	committed = true
	if result == 0 {
		return fmt.Errorf("commit branding resources: %w", err)
	}
	if len(symbols) > 0 {
		return restoreSymbols(path, symbols, symbolCount)
	}
	return nil
}

// Windows resource editing can discard the COFF overlay without fixing its header.
func restoreSymbols(path string, symbols []byte, count uint32) error {
	file, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer file.Close()
	var dos [64]byte
	if _, err := file.ReadAt(dos[:], 0); err != nil {
		return err
	}
	peOffset := int64(binary.LittleEndian.Uint32(dos[60:]))
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if offset > 1<<32-1 {
		return errors.New("branded executable is too large")
	}
	if _, err := file.Write(symbols); err != nil {
		return err
	}
	var fields [8]byte
	binary.LittleEndian.PutUint32(fields[:], uint32(offset))
	binary.LittleEndian.PutUint32(fields[4:], count)
	if _, err := file.WriteAt(fields[:], peOffset+12); err != nil {
		return err
	}
	_, err = file.WriteAt([]byte{0, 0, 0, 0}, peOffset+24+64)
	return err
}
