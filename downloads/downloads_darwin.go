package downloads

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>

// Return a malloc/strdup'd UTF-8 path or NULL on failure.
// Caller must free() the returned pointer.
char* getDownloadsDir() {
    @autoreleasepool {
        NSArray *urls = [[NSFileManager defaultManager]
            URLsForDirectory:NSDownloadsDirectory inDomains:NSUserDomainMask];
        if ([urls count] > 0) {
            NSString *path = [[urls objectAtIndex:0] path];
            if (path == nil) return NULL;
            const char *cstr = [path fileSystemRepresentation];
            if (cstr == NULL) return NULL;
            return strdup(cstr);
        }
    }
    return NULL;
}
*/
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"unsafe"
)

func getDownloadsDir() (string, error) {
	cstr := C.getDownloadsDir()
	if cstr != nil {
		defer C.free(unsafe.Pointer(cstr))
		path := C.GoString(cstr)
		if path != "" {
			return path, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("could not determine Downloads folder")
	}
	return filepath.Join(home, "Downloads"), nil
}
