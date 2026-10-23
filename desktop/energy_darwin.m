#import <Cocoa/Cocoa.h>
extern void tdEnergyChanged(int);

void tdObserveEnergy(void *dashboard, void *markdown, void *settings) {
 dispatch_async(dispatch_get_main_queue(), ^{
  NSWindow *main = (__bridge NSWindow *)dashboard;
  NSWindow *preferences = (__bridge NSWindow *)settings;
  NSWindow *preview = (__bridge NSWindow *)markdown;
  CGDirectDisplayID displays[32]; uint32_t count = 0;
  BOOL anyAwake = YES;
  if (CGGetOnlineDisplayList(32, displays, &count) == kCGErrorSuccess && count) {
   anyAwake = NO;
   for (uint32_t i = 0; i < count; i++) if (!CGDisplayIsAsleep(displays[i])) anyAwake = YES;
  }
  __block BOOL screensAsleep = !anyAwake, systemAsleep = NO;
  __block int previous = -1;
  void (^publish)(void) = ^{
   BOOL awake = !screensAsleep && !systemAsleep;
   int mask = (awake ? 1 : 0) |
    (awake && (main.occlusionState & NSWindowOcclusionStateVisible) ? 2 : 0) |
    (awake && (preview.occlusionState & NSWindowOcclusionStateVisible) ? 4 : 0) |
    (awake && (preferences.occlusionState & NSWindowOcclusionStateVisible) ? 8 : 0);
   if (mask != previous) { previous = mask; tdEnergyChanged(mask); }
  };
  NSNotificationCenter *workspace = NSWorkspace.sharedWorkspace.notificationCenter;
  [workspace addObserverForName:NSWorkspaceScreensDidSleepNotification object:nil queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *n){ screensAsleep = YES; publish(); }];
  [workspace addObserverForName:NSWorkspaceScreensDidWakeNotification object:nil queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *n){ screensAsleep = NO; publish(); }];
  [workspace addObserverForName:NSWorkspaceWillSleepNotification object:nil queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *n){ systemAsleep = YES; screensAsleep = YES; publish(); }];
  [workspace addObserverForName:NSWorkspaceDidWakeNotification object:nil queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *n){ systemAsleep = NO; publish(); }];
  [NSNotificationCenter.defaultCenter addObserverForName:NSWindowDidChangeOcclusionStateNotification object:nil queue:NSOperationQueue.mainQueue usingBlock:^(NSNotification *n){ publish(); }];
  publish();
 });
}
