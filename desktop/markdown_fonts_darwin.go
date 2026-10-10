//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#include <Cocoa/Cocoa.h>
#include <stdlib.h>
static char *tdFontFamilies(void) {
 @autoreleasepool {
  NSArray *families = [[[NSFontManager sharedFontManager] availableFontFamilies] sortedArrayUsingSelector:@selector(localizedCaseInsensitiveCompare:)];
  NSData *data = [NSJSONSerialization dataWithJSONObject:families options:0 error:nil];
  return data ? strndup(data.bytes, data.length) : NULL;
 }
}
*/
import "C"
import (
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var fontOnce sync.Once
var markdownFonts []string

func installedMarkdownFonts() []string {
	fontOnce.Do(func() {
		application.InvokeSync(func() {
			p := C.tdFontFamilies()
			if p != nil {
				defer C.free(unsafe.Pointer(p))
				_ = json.Unmarshal([]byte(C.GoString(p)), &markdownFonts)
			}
		})
	})
	return markdownFonts
}
