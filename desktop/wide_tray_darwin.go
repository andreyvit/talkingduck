//go:build darwin && cgo

package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

static char tdMenuMonitorKey;

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

// Called off the main thread; the block follows Wails' queued image assignment.
static double tdWidenTray(void *handle, double width, double height, double padding) {
    __block double actual = -1;
    dispatch_sync(dispatch_get_main_queue(), ^{
    NSStatusItem *item = (NSStatusItem *)handle;
    if (![item isKindOfClass:[NSStatusItem class]] || !item.button.image) return;
    NSImage *image = [item.button.image copy];
    image.size = NSMakeSize(width, height);
    image.template = YES;
    item.button.image = image;
    item.button.imageScaling = NSImageScaleNone;
    item.button.imagePosition = NSImageOnly;
    item.button.toolTip = @"TalkingDuck — mock checkout status. Click for overview; right-click for menu.";
    [item.button setAccessibilityLabel:@"TalkingDuck mock checkout status"];
    item.length = width + padding; // Preserve the chosen layout’s horizontal breathing room.
    [image release];
    actual = item.button.image.size.width;
    if (item.length != width + padding || item.button.image.size.height != height) actual = -2;
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

// updateWideTray runs off the AppKit main thread. Wails enqueues image assignment
// on dispatch_get_main_queue, whereas InvokeSync uses another dispatch mechanism.
// Dispatching our resize synchronously onto that SAME queue establishes ordering.
func updateWideTray(tray *application.SystemTray, data []byte, width, height, padding int) error {
	var handle unsafe.Pointer
	err := application.InvokeSyncWithError(func() error {
		var err error
		handle, err = nativeTrayHandle(tray)
		return err
	})
	if err != nil {
		return err
	}
	tray.SetTemplateIcon(data)
	actual := float64(C.tdWidenTray(handle, C.double(width), C.double(height), C.double(padding)))
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
