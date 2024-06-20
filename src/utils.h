/** @file */

#ifndef UTILS_H
#define UTILS_H

typedef unsigned long size_t;
#include "module/pteditor.h"
#include "ptedit.h"

int is_normal_page(size_t entry);
size_t virt2Phys(void *target, size_t pid);

#if defined(__i386__) || defined(__x86_64__)
#define FIRST_LEVEL_ENTRIES 256 // only 256, because upper half is kernel
#elif defined(__aarch64__)
#define FIRST_LEVEL_ENTRIES 512
#endif

int is_present(size_t entry);
size_t *getMappedPML4Entries(size_t pid);

#endif // !UTILS_H
