#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import <Carbon/Carbon.h>
#import <IOKit/hidsystem/IOLLEvent.h>

static WKWebView *tdMarkdownWebView(NSView *view) {
    if ([view isKindOfClass:WKWebView.class]) return (WKWebView *)view;
    for (NSView *child in view.subviews) {
        WKWebView *web = tdMarkdownWebView(child);
        if (web) return web;
    }
    return nil;
}

// WindowRuntimeReady installs this once for the persistent Markdown window.
void tdObserveMarkdownShift(void *handle) {
    dispatch_async(dispatch_get_main_queue(), ^{
        NSWindow *window = (NSWindow *)handle;
        WKWebView *web = tdMarkdownWebView(window.contentView);
        if (!web) return;
        [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskFlagsChanged handler:^NSEvent *(NSEvent *event) {
            if (!window.isKeyWindow || (event.keyCode != kVK_Shift && event.keyCode != kVK_RightShift)) return event;
            NSEventModifierFlags flags = event.modifierFlags;
            unsigned mask = 0;
            if ((flags & NSEventModifierFlagShift) &&
                !(flags & (NSEventModifierFlagCommand | NSEventModifierFlagControl | NSEventModifierFlagOption))) {
                if (flags & NX_DEVICELSHIFTKEYMASK) mask |= 1;
                if (flags & NX_DEVICERSHIFTKEYMASK) mask |= 2;
                // Synthetic keyboards may only supply the aggregate Shift flag.
                if (!mask) mask = event.keyCode == kVK_Shift ? 1 : 2;
            }
            NSString *js = [NSString stringWithFormat:@"window.tdMarkdownShift?.(%u)", mask];
            [web evaluateJavaScript:js completionHandler:nil];
            return event;
        }];
    });
}
