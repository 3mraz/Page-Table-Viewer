#include "utils.h"
#include "module/pteditor.h"
#include "ptedit.h"
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int is_normal_page(size_t entry) {
#if defined(__i386__) || defined(__x86_64__)
  return !(entry & (1ull << PTEDIT_PAGE_BIT_PSE));
#elif defined(__aarch64__)
  return 1;
#endif
}

size_t virt2Phys(void *target, size_t pid) {
  size_t phys = 0;
  ptedit_entry_t entry = ptedit_resolve(target, pid);
  if (is_normal_page(entry.pd)) {
    phys = (ptedit_get_pfn(entry.pte) << 12) | (((size_t)target) & 0xfff);
  } else {
    phys = (ptedit_get_pfn(entry.pd) << 21) | (((size_t)target) & 0x1fffff);
  }
  return phys;
}

int is_present(size_t entry) {
#if defined(__i386__) || defined(__x86_64__)
  return entry & (1ull << PTEDIT_PAGE_BIT_PRESENT);
#elif defined(__aarch64__)
  return (entry & 3) == 3;
#endif
}

#if defined(__i386__) || defined(__x86_64__)
#define FIRST_LEVEL_ENTRIES 256 // only 256, because upper half is kernel
#elif defined(__aarch64__)
#define FIRST_LEVEL_ENTRIES 512
#endif

size_t *getMappedPML4Entries(size_t pid) {
  size_t root = ptedit_get_paging_root(pid);
  size_t pagesize = ptedit_get_pagesize();
  size_t *entries = (size_t *)malloc((root / pagesize) * sizeof(size_t));
  size_t *pml4 = (size_t *)malloc((root / pagesize) * sizeof(size_t));
  memset((void *)entries, 0, (root / pagesize) * sizeof(size_t));
  ptedit_read_physical_page(root / pagesize, (char *)pml4);

  int pml4i;
  for (pml4i = 0; pml4i < FIRST_LEVEL_ENTRIES; pml4i++) {
    size_t pml4_entry = pml4[pml4i];
    if (!is_present(pml4_entry))
      continue;
    // ptedit_print_entry(pml4_entry);
    entries[pml4i] = pml4_entry;
  }
  free(pml4);
  return entries;
}
