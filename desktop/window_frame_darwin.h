#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#import <CoreGraphics/CoreGraphics.h>

static NSString *tdScreenID(NSScreen *screen) {
    CGDirectDisplayID display = [screen.deviceDescription[@"NSScreenNumber"] unsignedIntValue];
    CFUUIDRef uuid = CGDisplayCreateUUIDFromDisplayID(display);
    if (!uuid) return @"";
    NSString *result = [(NSString *)CFUUIDCreateString(NULL, uuid) autorelease];
    CFRelease(uuid);
    return result;
}

static void tdSavePlacement(NSWindow *window, NSString *key) {
    if (!window.visible || !window.screen || (window.styleMask & NSWindowStyleMaskFullScreen)) return;
    NSRect frame = window.frame, screen = window.screen.visibleFrame;
    frame.origin.x -= screen.origin.x;
    frame.origin.y -= screen.origin.y;
    [[NSUserDefaults standardUserDefaults] setObject:@{
        @"screen": tdScreenID(window.screen), @"frame": NSStringFromRect(frame)
    } forKey:key];
}

static int tdRememberWindowFrame(void *handle, const char *name) {
    __block BOOL saved = NO;
    void (^configure)(void) = ^{
        NSWindow *window = (NSWindow *)handle;
        NSString *key = [NSString stringWithUTF8String:name];
        NSString *placementKey = [@"TalkingDuck Placement " stringByAppendingString:key];
        NSDictionary *placement = [[NSUserDefaults standardUserDefaults] dictionaryForKey:placementKey];
        // AppKit's autosaved tiling metadata can replace the exact frame with an
        // old untiled size when the window first becomes visible. Only use it
        // to migrate installations without our display-aware placement.
        if ([placement[@"frame"] isKindOfClass:[NSString class]]) {
            NSScreen *target = nil;
            for (NSScreen *screen in NSScreen.screens) {
                if ([tdScreenID(screen) isEqual:placement[@"screen"]]) { target = screen; break; }
            }
            // A disconnected display must not leave the window offscreen.
            if (!target) target = NSScreen.mainScreen;
            NSRect visible = target.visibleFrame;
            NSRect frame = NSRectFromString(placement[@"frame"]);
            if (frame.size.width > 0 && frame.size.height > 0) {
                frame.size.width = MIN(frame.size.width, visible.size.width);
                frame.size.height = MIN(frame.size.height, visible.size.height);
                frame.origin.x = visible.origin.x + MAX(0, MIN(frame.origin.x, visible.size.width-frame.size.width));
                frame.origin.y = visible.origin.y + MAX(0, MIN(frame.origin.y, visible.size.height-frame.size.height));
                [window setFrame:frame display:NO];
            }
        } else {
            [window setFrameUsingName:key force:YES];
        }
        saved = YES;
        for (NSString *event in @[NSWindowDidMoveNotification, NSWindowDidResizeNotification, NSWindowDidBecomeKeyNotification]) {
            // Windows live for the app lifetime; notification blocks share that lifetime.
            [[NSNotificationCenter defaultCenter] addObserverForName:event object:window queue:nil usingBlock:^(NSNotification *note) {
                tdSavePlacement(window, placementKey);
            }];
        }
    };
    if ([NSThread isMainThread]) configure();
    else dispatch_sync(dispatch_get_main_queue(), configure);
    return saved;
}
