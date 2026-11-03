#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>
#import "pdf_pages_darwin.h"

@interface TDPDFJob : NSObject<WKNavigationDelegate>
@property(retain) WKWebView *view;
@property(retain) NSURL *url;
@property(copy) NSString *markerHost;
@property BOOL dark;
@property(copy) void (^completion)(BOOL);
@end

@implementation TDPDFJob
- (void)finish:(BOOL)ok {
    if (!self.completion) return;
    [NSObject cancelPreviousPerformRequestsWithTarget:self];
    self.view.navigationDelegate=nil;
    [self.view stopLoading];
    self.view=nil;
    self.completion(ok);
    self.completion=nil;
}
- (void)webView:(WKWebView *)view didFinishNavigation:(WKNavigation *)nav {
    [NSObject cancelPreviousPerformRequestsWithTarget:self];
    NSURL *url=self.url;
    NSPrintInfo *info = [[NSPrintInfo.sharedPrintInfo copy] autorelease];
    info.paperSize = NSMakeSize(595,842);
    info.leftMargin = info.rightMargin = 0;
    info.topMargin = info.bottomMargin = 28;
    info.horizontalPagination = NSPrintingPaginationModeFit;
    info.verticalPagination = NSPrintingPaginationModeAutomatic;
    info.verticallyCentered = NO;
    info.jobDisposition = NSPrintSaveJob;
    info.dictionary[NSPrintJobSavingURL] = url;
    NSPrintOperation *op = [self.view printOperationWithPrintInfo:info];
    op.showsPrintPanel = NO;
    op.showsProgressPanel = NO;
    op.canSpawnSeparateThread = YES;
    [op runOperationModalForWindow:NSApp.keyWindow delegate:self
        didRunSelector:@selector(printOperation:finished:contextInfo:) contextInfo:NULL];
}
- (void)printOperation:(NSPrintOperation *)op finished:(BOOL)success contextInfo:(void *)context {
    // Let AppKit finish tearing down its print operation before releasing the view.
    dispatch_async(dispatch_get_main_queue(), ^{
        PDFDocument *pdf = success ? [[[PDFDocument alloc] initWithURL:self.url] autorelease] : nil;
        [self finish:tdFinishPages(pdf,self.url,self.markerHost,self.dark)];
    });
}
- (void)timeout { [self finish:NO]; }
- (void)webView:(WKWebView *)view didFailNavigation:(WKNavigation *)nav withError:(NSError *)err { [self finish:NO]; }
- (void)webView:(WKWebView *)view didFailProvisionalNavigation:(WKNavigation *)nav withError:(NSError *)err { [self finish:NO]; }
- (void)webViewWebContentProcessDidTerminate:(WKWebView *)view { [self finish:NO]; }
- (void)dealloc { [_markerHost release]; [_view release]; [_url release]; [_completion release]; [super dealloc]; }
@end

int tdWritePDF(const char *html,const char *dest,const char *host,int dark) {
    dispatch_semaphore_t done=dispatch_semaphore_create(0);
    __block BOOL saved=NO;
    TDPDFJob *job=[TDPDFJob new];
    dispatch_async(dispatch_get_main_queue(), ^{
        WKWebViewConfiguration *config=[[[WKWebViewConfiguration alloc] init] autorelease];
        config.websiteDataStore=WKWebsiteDataStore.nonPersistentDataStore;
        config.defaultWebpagePreferences.allowsContentJavaScript=NO;
        if (@available(macOS 13.3,*)) config.preferences.shouldPrintBackgrounds=YES;
        job.dark=dark;
        job.markerHost=[NSString stringWithUTF8String:host];
        job.url=[NSURL fileURLWithPath:[NSString stringWithUTF8String:dest]];
        job.view=[[[WKWebView alloc] initWithFrame:NSMakeRect(0,0,794,1123) configuration:config] autorelease];
        job.view.navigationDelegate=job;
        job.completion=^(BOOL ok){ saved=ok; dispatch_semaphore_signal(done); };
        [job performSelector:@selector(timeout) withObject:nil afterDelay:45];
        [job.view loadHTMLString:[NSString stringWithUTF8String:html] baseURL:nil];
    });
    dispatch_semaphore_wait(done,DISPATCH_TIME_FOREVER);
    dispatch_async(dispatch_get_main_queue(), ^{[job release];});
    dispatch_release(done);
    return saved;
}
