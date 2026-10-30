//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

static char tdMenuMonitorKey;
static char tdTrayDiagnosticKey;

// Consume right-down before NSStatusBarButton enters its own tracking loop.
// Keep the Wails-created menu and its item callbacks, but avoid showMenu's
// synthetic mouseDown and temporary statusItem.menu attachment.
static void tdInstallMenuMonitor(void *handle, void *menuHandle) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSStatusItem *item = (NSStatusItem *)handle;
        NSMenu *menu = (NSMenu *)menuHandle;
        id previous = objc_getAssociatedObject(item, &tdMenuMonitorKey);
        if (previous) [NSEvent removeMonitor:previous];
        id monitor = [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskRightMouseDown
            handler:^NSEvent *(NSEvent *event) {
                NSStatusBarButton *button = item.button;
                if (event.window != button.window) return event;
                NSPoint point = [button convertPoint:event.locationInWindow fromView:nil];
                if (!NSPointInRect(point, button.bounds)) return event;
                NSLog(@"TalkingDuck tray: native right-click menu open");
                [button highlight:YES];
                @try {
                    [menu popUpMenuPositioningItem:nil
                        atLocation:NSMakePoint(NSMinX(button.bounds), NSMinY(button.bounds))
                        inView:button];
                } @finally {
                    [button highlight:NO];
                    NSLog(@"TalkingDuck tray: native right-click menu closed");
                }
                return nil;
            }];
        objc_setAssociatedObject(item, &tdMenuMonitorKey, monitor, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    });
}

// Decode, assign, and size in one main-queue operation.
static double tdWidenTray(void *handle, const void *bytes, int length, double width, double height, double padding, double prefix, int tintPing) {
    __block double actual = -1;
    dispatch_sync(dispatch_get_main_queue(), ^{
    NSStatusItem *item = (NSStatusItem *)handle;
    if (![item isKindOfClass:[NSStatusItem class]]) return;
    NSData *data = [NSData dataWithBytes:bytes length:length];
    NSImage *image = [[NSImage alloc] initWithData:data];
    if (!image) return;
    image.size = NSMakeSize(width, height);
    if (prefix > 0) {
        NSImage *colored = [[NSImage alloc] initWithSize:NSMakeSize(width, height)];
        [colored lockFocus];
        [NSGraphicsContext currentContext].imageInterpolation = NSImageInterpolationNone;
        [NSGraphicsContext currentContext].shouldAntialias = NO;
        [image drawInRect:NSMakeRect(0, 0, width, height)];
        [item.button.effectiveAppearance performAsCurrentDrawingAppearance:^{
            [NSColor.labelColor setFill];
            NSRectFillUsingOperation(NSMakeRect(prefix, 0, width-prefix, height), NSCompositingOperationSourceAtop);
            if (tintPing) NSRectFillUsingOperation(NSMakeRect(0, 0, 8, height), NSCompositingOperationSourceAtop);
        }];
        [colored unlockFocus];
        [image release]; image=colored; image.template=NO;
    } else { image.template=YES; }
    item.button.image = image;
    item.button.imageScaling = NSImageScaleNone;
    item.button.imagePosition = NSImageOnly;
    item.button.toolTip = @"TalkingDuck — checkout status. Click for overview; right-click for menu.";
    [item.button setAccessibilityLabel:@"TalkingDuck checkout status"];
    item.length = width + padding; // Preserve the chosen layout’s horizontal breathing room.
    [image release];
    actual = item.button.image.size.width;
    if (item.length != width + padding || item.button.image.size.height != height) actual = -2;
    NSString *diagnostic = [NSString stringWithFormat:@"visible=%d windowVisible=%d occluded=%d length=%.0f image=%@ frame=%@ screen=%@", item.visible, item.button.window.visible, !(item.button.window.occlusionState & NSWindowOcclusionStateVisible), item.length, NSStringFromSize(item.button.image.size), NSStringFromRect(item.button.window.frame), NSStringFromRect(item.button.window.screen.frame)];
    if (![diagnostic isEqual:objc_getAssociatedObject(item, &tdTrayDiagnosticKey)]) {
        objc_setAssociatedObject(item, &tdTrayDiagnosticKey, diagnostic, OBJC_ASSOCIATION_COPY_NONATOMIC);
        NSLog(@"TalkingDuck tray: %@", diagnostic);
    }
    });
    return actual;
}
*/
import "C"

import (
	"fmt"
	"reflect"
	"runtime/debug"
	"unsafe"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const wailsVersion = "v3.0.0-beta.27"

// nativeTrayHandle is the only dependency on Wails internals. It reads named
// fields, not hard-coded memory offsets. Pin/check the version and fail closed
// if Wails changes its layout. Must run on the application main thread after Run.
func nativeTrayHandle(tray *application.SystemTray) (unsafe.Pointer, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("missing build metadata")
	}
	matched := false
	for _, dep := range info.Deps {
		if dep.Path == "github.com/wailsapp/wails/v3" {
			matched = dep.Version == wailsVersion && dep.Replace == nil
		}
	}
	if !matched {
		return nil, fmt.Errorf("native tray shim requires unmodified Wails %s", wailsVersion)
	}
	impl := reflect.ValueOf(tray).Elem().FieldByName("impl")
	if !impl.IsValid() || impl.Kind() != reflect.Interface || impl.IsNil() {
		return nil, fmt.Errorf("Wails tray has not started")
	}
	native := impl.Elem()
	if native.Kind() != reflect.Pointer || native.IsNil() || native.Type().Elem().Name() != "macosSystemTray" {
		return nil, fmt.Errorf("unexpected Wails tray implementation")
	}
	field := native.Elem().FieldByName("nsStatusItem")
	if !field.IsValid() || field.Kind() != reflect.UnsafePointer || field.IsNil() {
		return nil, fmt.Errorf("Wails native status item unavailable")
	}
	return field.UnsafePointer(), nil
}

// updateWideTray runs off the AppKit main thread. Do not call Wails image setters:
// their run-loop callback can enqueue a square image after our sizing operation.
type trayMonitorStyle struct {
	Width    int
	TintPing bool
}

func updateWideTray(tray *application.SystemTray, data []byte, width, height, padding int, monitorOptions ...trayMonitorStyle) error {
	var handle unsafe.Pointer
	err := application.InvokeSyncWithError(func() error {
		var err error
		handle, err = nativeTrayHandle(tray)
		return err
	})
	if err != nil {
		return err
	}
	prefix, tintPing := 0, false
	if len(monitorOptions) > 0 {
		prefix = monitorOptions[0].Width
		tintPing = monitorOptions[0].TintPing
	}

	tint := 0
	if tintPing {
		tint = 1
	}
	if len(data) == 0 {
		return fmt.Errorf("empty tray image")
	}
	bytes := C.CBytes(data)
	defer C.free(bytes)
	actual := float64(C.tdWidenTray(handle, bytes, C.int(len(data)), C.double(width), C.double(height), C.double(padding), C.double(prefix), C.int(tint)))
	if actual != float64(width) {
		return fmt.Errorf("native image width %.0f; expected %d", actual, width)
	}
	return nil
}

// installTrayMenu runs after Wails has created the status item and native menu.
func installTrayMenu(tray *application.SystemTray) error {
	var handle, menu unsafe.Pointer
	err := application.InvokeSyncWithError(func() error {
		var err error
		handle, err = nativeTrayHandle(tray)
		if err != nil {
			return err
		}
		native := reflect.ValueOf(tray).Elem().FieldByName("impl").Elem().Elem()
		field := native.FieldByName("nsMenu")
		if !field.IsValid() || field.Kind() != reflect.UnsafePointer || field.IsNil() {
			return fmt.Errorf("Wails native tray menu unavailable")
		}
		menu = field.UnsafePointer()
		return nil
	})
	if err != nil {
		return err
	}
	C.tdInstallMenuMonitor(handle, menu)
	return nil
}
