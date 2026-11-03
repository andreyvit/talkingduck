//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa -framework UniformTypeIdentifiers
#import <Cocoa/Cocoa.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#include <stdlib.h>


static int tdCopyImage(const void *bytes, size_t length, const char *mime, int png) {
 @autoreleasepool {
    NSData *data=[NSData dataWithBytes:bytes length:length];
    NSString *type=[UTType typeWithMIMEType:[NSString stringWithUTF8String:mime]].identifier;
    if (png && ![type isEqualToString:NSPasteboardTypePNG]) {
        NSImage *image=[[[NSImage alloc] initWithData:data] autorelease];
        if (!image || image.size.width<=0 || image.size.height<=0 ||
            image.size.width*image.size.height>64*1024*1024) return NO;
        NSBitmapImageRep *bitmap=[NSBitmapImageRep imageRepWithData:image.TIFFRepresentation];
        data=[bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        type=NSPasteboardTypePNG;
    }
    if (!data || !type) return NO;
    NSPasteboardItem *item=[[[NSPasteboardItem alloc] init] autorelease];
    if (![item setData:data forType:type]) return NO;
    NSPasteboard *pb=NSPasteboard.generalPasteboard;
    [pb clearContents];
    return [pb writeObjects:@[item]];
 }
}
*/
import "C"
import (
	"encoding/base64"
	"fmt"

	"io"
	"net/http"
	"os/exec"
	"strings"
	"talkingduck/markdownview"
	"time"
	"unsafe"
)

func openImageFile(path string, reveal bool) error {
	if path == "" {
		return fmt.Errorf("image has no local path")
	}
	args := []string{path}
	if reveal {
		args = []string{"-R", path}
	}
	return exec.Command("/usr/bin/open", args...).Run()
}
func copyImage(img markdownview.Image, format string) error {
	var mime string
	var data []byte
	var err error
	if strings.HasPrefix(img.Src, "data:") {
		header, encoded, ok := strings.Cut(img.Src, ",")
		if !ok || !strings.HasSuffix(header, ";base64") {
			return fmt.Errorf("unsupported image encoding")
		}
		mime = strings.TrimSuffix(strings.TrimPrefix(header, "data:"), ";base64")
		data, err = base64.StdEncoding.DecodeString(encoded)
	} else {
		client := http.Client{Timeout: 15 * time.Second}
		response, e := client.Get(img.Src)
		if e != nil {
			return e
		}
		defer response.Body.Close()
		if response.StatusCode/100 != 2 {
			return fmt.Errorf("image server returned %s", response.Status)
		}
		data, err = io.ReadAll(io.LimitReader(response.Body, markdownview.MaxImage+1))
		mime, _, _ = strings.Cut(response.Header.Get("Content-Type"), ";")
	}
	if err != nil {
		return err
	}
	if len(data) == 0 || len(data) > markdownview.MaxImage {
		return fmt.Errorf("image is empty or exceeds the copy limit")
	}
	mt := C.CString(mime)
	defer C.free(unsafe.Pointer(mt))
	var png C.int
	if format == "PNG" {
		png = 1
	}
	if C.tdCopyImage(unsafe.Pointer(&data[0]), C.size_t(len(data)), mt, png) == 0 {
		return fmt.Errorf("macOS could not copy the image as %s", format)
	}
	return nil
}
