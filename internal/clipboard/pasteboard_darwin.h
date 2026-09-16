#ifndef IMOOGI_PASTEBOARD_DARWIN_H
#define IMOOGI_PASTEBOARD_DARWIN_H

#include <stddef.h>

long imoogi_pasteboard_change_count(void);
char *imoogi_pasteboard_inspect_json(void);
char *imoogi_pasteboard_files_json(void);
unsigned char *imoogi_pasteboard_png(size_t *length);
void imoogi_pasteboard_free(void *pointer);

#endif
