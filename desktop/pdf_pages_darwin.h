#import <PDFKit/PDFKit.h>
#import <CoreText/CoreText.h>

// Paint each sheet in one pass, keeping text/vector artwork and PDF links.
static BOOL tdFinishPages(PDFDocument *pdf, NSURL *dest, NSString *host, BOOL dark) {
    if (!pdf.pageCount) return NO;
    NSMutableDictionary<NSString *,NSNumber *> *numbers = [NSMutableDictionary dictionary];
    NSMutableDictionary<NSNumber *, NSMutableDictionary<NSString *, NSValue *> *> *anchors = [NSMutableDictionary dictionary];
    for (NSUInteger i=0; i<pdf.pageCount; i++) {
        for (PDFAnnotation *link in [pdf pageAtIndex:i].annotations) {
            PDFDestination *d=link.destination;
            if (d.page) {
                NSNumber *pageIndex=@([pdf indexForPage:d.page]);
                if(!anchors[pageIndex]) anchors[pageIndex]=[NSMutableDictionary dictionary];
                NSString *name=[NSString stringWithFormat:@"anchor-%lu-%.3f-%.3f",pageIndex.unsignedLongValue,d.point.x,d.point.y];
                anchors[pageIndex][name]=[NSValue valueWithPoint:d.point];
            }
            NSURL *url=link.URL;
            if ([url.host isEqual:host] && [url.path hasPrefix:@"/heading/"])
                numbers[url.lastPathComponent]=@(i+1);
        }
    }
    NSMutableData *data=[NSMutableData data];
    CGDataConsumerRef consumer=CGDataConsumerCreateWithCFData((CFMutableDataRef)data);
    CGContextRef ctx=CGPDFContextCreate(consumer,NULL,(CFDictionaryRef)pdf.documentAttributes);
    CGDataConsumerRelease(consumer);
    if (!ctx) return NO;
    NSMutableSet *numbered=[NSMutableSet set];
    BOOL valid=YES;
    for (NSUInteger i=0;i<pdf.pageCount;i++) { @autoreleasepool {
        PDFPage *page=[pdf pageAtIndex:i];
        CGRect box=[page boundsForBox:kPDFDisplayBoxMediaBox];
        NSData *media=[NSData dataWithBytes:&box length:sizeof(box)];
        CGPDFContextBeginPage(ctx,(CFDictionaryRef)@{(NSString *)kCGPDFContextMediaBox:media});
        if (dark) CGContextSetRGBFillColor(ctx,13./255,17./255,23./255,1);
        else CGContextSetGrayFillColor(ctx,1,1);
        CGContextFillRect(ctx,box);
        CGContextDrawPDFPage(ctx,page.pageRef);
        for (NSString *name in anchors[@(i)]) CGPDFContextAddDestinationAtPoint(ctx,(CFStringRef)name,anchors[@(i)][name].pointValue);
        for (PDFAnnotation *link in page.annotations) {
            NSURL *url=link.URL;
            CGRect r=link.bounds;
            if (![url.host isEqual:host]) {
                if(url) CGPDFContextSetURLForRect(ctx,(CFURLRef)url,r);
                else if(link.destination.page) {
                    // Preserve in-document links and their scroll positions.
                    PDFDestination *d=link.destination;
                    NSString *name=[NSString stringWithFormat:@"anchor-%lu-%.3f-%.3f",(unsigned long)[pdf indexForPage:d.page],d.point.x,d.point.y];
                    CGPDFContextSetDestinationForRect(ctx,(CFStringRef)name,r);
                }
                continue;
            }
            NSString *key=url.lastPathComponent;
            if ([url.path hasPrefix:@"/heading/"]) {
                CGPDFContextAddDestinationAtPoint(ctx,(CFStringRef)key,CGPointMake(0,MIN(CGRectGetMaxY(box),CGRectGetMaxY(r)+18)));
            } else if ([url.path hasPrefix:@"/entry/"]) {
                NSNumber *number=numbers[key];
                if(!number){valid=NO;continue;}
                CGPDFContextSetDestinationForRect(ctx,(CFStringRef)key,r);
                if([numbered containsObject:key])continue;
                [numbered addObject:key];
                CGFloat size=MAX(8,MIN(13,r.size.height*.82));
                NSDictionary *attrs=@{NSFontAttributeName:[NSFont monospacedDigitSystemFontOfSize:size weight:NSFontWeightRegular],
                    NSForegroundColorAttributeName:dark ? NSColor.whiteColor : NSColor.blackColor};
                NSAttributedString *str=[[[NSAttributedString alloc] initWithString:number.stringValue attributes:attrs] autorelease];
                CTLineRef line=CTLineCreateWithAttributedString((CFAttributedStringRef)str);
                CGFloat x=CGRectGetMaxX(box)-24-CTLineGetTypographicBounds(line,NULL,NULL,NULL);
                CGContextSetTextMatrix(ctx,CGAffineTransformIdentity);
                CGContextSetTextPosition(ctx,x,CGRectGetMinY(r)+size*.15);
                CTLineDraw(line,ctx);CFRelease(line);
                CGPDFContextSetDestinationForRect(ctx,(CFStringRef)key,CGRectMake(x-4,r.origin.y,CGRectGetMaxX(box)-x,r.size.height));
            }
        }
        CGPDFContextEndPage(ctx);
    }}
    CGPDFContextClose(ctx);CGContextRelease(ctx);
    return valid && numbered.count==numbers.count && [data writeToURL:dest options:NSDataWritingAtomic error:nil];
}
