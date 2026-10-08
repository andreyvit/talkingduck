//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include "window_frame_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Call after Wails creates and positions the native window.
func rememberWindowFrame(window *application.WebviewWindow, name string) error {
	handle := window.NativeWindow()
	if handle == nil {
		return fmt.Errorf("native window unavailable: %s", name)
	}
	key := C.CString(name)
	defer C.free(unsafe.Pointer(key))
	if C.tdRememberWindowFrame(handle, key) == 0 {
		return fmt.Errorf("cannot autosave window frame: %s", name)
	}
	return nil
}
