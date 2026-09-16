#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>
#include "pasteboard_darwin.h"

static char *copy_json(id value) {
  NSError *error = nil;
  NSData *data = [NSJSONSerialization dataWithJSONObject:value options:0 error:&error];
  if (error != nil || data == nil) return NULL;
  char *result = malloc([data length] + 1);
  if (result == NULL) return NULL;
  memcpy(result, [data bytes], [data length]);
  result[[data length]] = '\0';
  return result;
}

long imoogi_pasteboard_change_count(void) {
  @autoreleasepool { return (long)[[NSPasteboard generalPasteboard] changeCount]; }
}

char *imoogi_pasteboard_inspect_json(void) {
  @autoreleasepool {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    NSArray<NSPasteboardType> *types = [pasteboard types] ?: @[];
    NSMutableArray<NSString *> *values = [NSMutableArray arrayWithCapacity:[types count]];
    for (NSPasteboardType type in types) [values addObject:(NSString *)type];
    return copy_json(values);
  }
}

char *imoogi_pasteboard_files_json(void) {
  @autoreleasepool {
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    NSDictionary *options = @{NSPasteboardURLReadingFileURLsOnlyKey: @YES};
    NSArray<NSURL *> *urls = [pasteboard readObjectsForClasses:@[[NSURL class]] options:options] ?: @[];
    NSMutableArray<NSString *> *paths = [NSMutableArray arrayWithCapacity:[urls count]];
    for (NSURL *url in urls) if ([url isFileURL] && [url path] != nil) [paths addObject:[url path]];
    return copy_json(paths);
  }
}

unsigned char *imoogi_pasteboard_png(size_t *length) {
  @autoreleasepool {
    *length = 0;
    NSPasteboard *pasteboard = [NSPasteboard generalPasteboard];
    NSData *data = [pasteboard dataForType:NSPasteboardTypePNG];
    if (data == nil) {
      NSData *tiff = [pasteboard dataForType:NSPasteboardTypeTIFF];
      if (tiff != nil) {
        NSBitmapImageRep *image = [NSBitmapImageRep imageRepWithData:tiff];
        data = [image representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
      }
    }
    if (data == nil || [data length] == 0) return NULL;
    unsigned char *result = malloc([data length]);
    if (result == NULL) return NULL;
    memcpy(result, [data bytes], [data length]);
    *length = [data length];
    return result;
  }
}

void imoogi_pasteboard_free(void *pointer) { free(pointer); }
