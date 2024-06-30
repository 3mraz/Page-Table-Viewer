/** @file */

#ifndef UTILS_H
#define UTILS_H

typedef unsigned long size_t;

typedef struct {
  size_t entry;
  size_t vaddr;
} PTEntry;
#include "module/pteditor.h"
#include "ptedit.h"

int is_normal_page(size_t entry);
size_t virt_2_phys(void *target, size_t pid);

#if defined(__i386__) || defined(__x86_64__)
#define FIRST_LEVEL_ENTRIES 256 // only 256, because upper half is kernel
#elif defined(__aarch64__)
#define FIRST_LEVEL_ENTRIES 512
#endif

int is_present(size_t entry);
int bit_set(size_t entry, size_t bit);
PTEntry *get_mapped_PML4_entries(size_t pid);
PTEntry *get_mapped_PDPT_entries(size_t pid, size_t pml4i);
PTEntry *get_mapped_PD_entries(size_t pid, size_t pml4i, size_t pdpti);
PTEntry *get_PTE_entries(size_t pid, size_t pml4i, size_t pdpti, size_t pdi);
#endif // !UTILS_H
