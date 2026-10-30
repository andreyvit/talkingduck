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
 NSString *suite = [@"TalkingDuckMigrationTest." stringByAppendingString:NSUUID.UUID.UUIDString];
 NSUserDefaults *migration = [[NSUserDefaults alloc] initWithSuiteName:suite];
 NSDictionary *legacy = @{@"TalkingDuck Placement Inventory": @{@"frame": @"old"},
                          @"NSWindow Frame Inventory": @"legacy frame",
                          @"NSStatusItem Visible Item-0": @NO};
 tdMigrateWindowPreferences(migration, legacy, @"Inventory");
 NSCAssert([[migration dictionaryForKey:@"TalkingDuck Placement Inventory"] isEqual:legacy[@"TalkingDuck Placement Inventory"]], @"Migrate placement");
 NSCAssert([[migration stringForKey:@"NSWindow Frame Inventory"] isEqual:@"legacy frame"], @"Migrate legacy frame");
 NSCAssert(![migration objectForKey:@"NSStatusItem Visible Item-0"], @"Do not migrate tray state");
 [migration setObject:@{@"frame": @"new"} forKey:@"TalkingDuck Placement Inventory"];
 tdMigrateWindowPreferences(migration, legacy, @"Inventory");
 NSCAssert([[migration dictionaryForKey:@"TalkingDuck Placement Inventory"][@"frame"] isEqual:@"new"], @"Keep current placement");
 [migration removePersistentDomainForName:suite];
 NSLog(@"PASS bundle preference migration");
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
