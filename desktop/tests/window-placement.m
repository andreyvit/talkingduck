#import "../window_frame_darwin.h"

// Catch accidental reintroduction of AppKit's competing tiling restoration.
@interface PlacementTestWindow : NSWindow
@end
@implementation PlacementTestWindow
- (BOOL)setFrameAutosaveName:(NSString *)name {
    if (name.length) { NSLog(@"FAIL registered competing frame autosave"); exit(1); }
    return [super setFrameAutosaveName:name];
}
- (BOOL)setFrameUsingName:(NSString *)name force:(BOOL)force {
    NSLog(@"FAIL replayed legacy tiling metadata despite exact placement"); exit(1);
}
@end

int main() { @autoreleasepool {
 [NSApplication sharedApplication];
 NSScreen *screen = NSScreen.screens.lastObject;
 NSString *name = @"TalkingDuckPlacementTest";
 NSString *key = [@"TalkingDuck Placement " stringByAppendingString:name];
 NSUserDefaults *defaults = NSUserDefaults.standardUserDefaults;
 for (int missing=0; missing<2; missing++) {
  [defaults setObject:@{@"screen": missing ? @"disconnected" : tdScreenID(screen), @"frame": NSStringFromRect(NSMakeRect(40,50,600,500))} forKey:key];
  NSWindow *window = [[PlacementTestWindow alloc] initWithContentRect:NSMakeRect(0,0,1280,720) styleMask:NSWindowStyleMaskTitled|NSWindowStyleMaskResizable backing:NSBackingStoreBuffered defer:NO];
  tdRememberWindowFrame(window, name.UTF8String);
  NSScreen *target = missing ? NSScreen.mainScreen : screen;
  NSRect expected = NSMakeRect(target.visibleFrame.origin.x+40,target.visibleFrame.origin.y+50,600,500);
  if (!NSEqualRects(window.frame,expected)) { NSLog(@"FAIL %@ != %@",NSStringFromRect(window.frame),NSStringFromRect(expected)); return 1; }
  NSLog(@"PASS %@ %@", missing?@"disconnected fallback":@"secondary screen", NSStringFromRect(window.frame));
  [window setFrameAutosaveName:@""];
 }
 [defaults removeObjectForKey:key];
 [NSWindow removeFrameUsingName:name];
 return 0;
}}
