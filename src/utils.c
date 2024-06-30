#include "utils.h"
#include "module/pteditor.h"
#include "ptedit.h"
#include <stdlib.h>
#include <string.h>

#define FIRST_LEVEL_ENTRIES 256

int is_normal_page(size_t entry) {
#if defined(__i386__) || defined(__x86_64__)
  return !(entry & (1ull << PTEDIT_PAGE_BIT_PSE));
#elif defined(__aarch64__)
  return 1;
#endif
}

size_t num_entries_per_lvl() {
  size_t page_size = ptedit_get_pagesize();
  return page_size / sizeof(size_t);
}

size_t virt_2_phys(void *target, size_t pid) {
  size_t phys = 0;
  ptedit_entry_t entry = ptedit_resolve(target, pid);
  if (is_normal_page(entry.pd)) {
    phys = (ptedit_get_pfn(entry.pte) << 12) | (((size_t)target) & 0xfff);
  } else {
    phys = (ptedit_get_pfn(entry.pd) << 21) | (((size_t)target) & 0x1fffff);
  }
  return phys;
}

int bit_set(size_t entry, size_t bit) {
  return (!!((entry) & (1ull << (bit))));
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

PTEntry *get_mapped_PML4_entries(size_t pid) {
  size_t root = ptedit_get_paging_root(pid);
  size_t pagesize = ptedit_get_pagesize();
  PTEntry *entries = (PTEntry *)malloc(num_entries_per_lvl() * sizeof(PTEntry));
  size_t *pml4 = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  memset((void *)entries, 0, num_entries_per_lvl() * sizeof(PTEntry));
  ptedit_read_physical_page(root / pagesize, (char *)pml4);

  size_t pml4i;
  for (pml4i = 0; pml4i < FIRST_LEVEL_ENTRIES; pml4i++) {
    PTEntry pml4_entry;
    pml4_entry.entry = pml4[pml4i];
    pml4_entry.vaddr = pml4i << 39;
    if (!is_present(pml4_entry.entry))
      continue;
    entries[pml4i] = pml4_entry;
  }
  free(pml4);
  return entries;
}

PTEntry *get_mapped_PDPT_entries(size_t pid, size_t pml4i) {
  size_t root = ptedit_get_paging_root(pid);
  size_t pagesize = ptedit_get_pagesize();
  PTEntry *entries = (PTEntry *)malloc(num_entries_per_lvl() * sizeof(PTEntry));
  size_t *pdpt = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pml4 = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  memset((void *)entries, 0, num_entries_per_lvl() * sizeof(PTEntry));
  ptedit_read_physical_page(root / pagesize, (char *)pml4);
  size_t pml4_entry = pml4[pml4i];
  ptedit_read_physical_page(ptedit_get_pfn(pml4_entry), (char *)pdpt);

  size_t pdpti;
  for (pdpti = 0; pdpti < 512; pdpti++) {
    PTEntry pdpt_entry;
    pdpt_entry.entry = pdpt[pdpti];
    pdpt_entry.vaddr = (pml4i << 39) | (pdpti << 30);
    if (!is_present(pdpt_entry.entry))
      continue;
    entries[pdpti] = pdpt_entry;
  }
  free(pml4);
  free(pdpt);
  return entries;
}

PTEntry *get_mapped_PD_entries(size_t pid, size_t pml4i, size_t pdpti) {
  size_t root = ptedit_get_paging_root(pid);
  size_t pagesize = ptedit_get_pagesize();
  PTEntry *entries = (PTEntry *)malloc(num_entries_per_lvl() * sizeof(PTEntry));
  size_t *pd = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pdpt = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pml4 = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  memset((void *)entries, 0, num_entries_per_lvl() * sizeof(PTEntry));
  ptedit_read_physical_page(root / pagesize, (char *)pml4);
  size_t pml4_entry = pml4[pml4i];
  ptedit_read_physical_page(ptedit_get_pfn(pml4_entry), (char *)pdpt);
  size_t pdpt_entry = pdpt[pdpti];
  ptedit_read_physical_page(ptedit_get_pfn(pdpt_entry), (char *)pd);

  size_t pdi;
  for (pdi = 0; pdi < 512; pdi++) {
    PTEntry pd_entry;
    pd_entry.entry = pd[pdi];
    pd_entry.vaddr = (pml4i << 39) | (pdpti << 30) | (pdi << 21);
    if (!is_present(pd_entry.entry))
      continue;
    entries[pdi] = pd_entry;
  }
  free(pd);
  free(pml4);
  free(pdpt);
  return entries;
}
PTEntry *get_PTE_entries(size_t pid, size_t pml4i, size_t pdpti, size_t pdi) {
  size_t root = ptedit_get_paging_root(pid);
  size_t pagesize = ptedit_get_pagesize();
  PTEntry *entries = (PTEntry *)malloc(num_entries_per_lvl() * sizeof(PTEntry));
  size_t *pte = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pd = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pdpt = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  size_t *pml4 = (size_t *)malloc(num_entries_per_lvl() * sizeof(size_t));
  memset((void *)entries, 0, num_entries_per_lvl() * sizeof(PTEntry));

  ptedit_read_physical_page(root / pagesize, (char *)pml4);
  size_t pml4_entry = pml4[pml4i];
  ptedit_read_physical_page(ptedit_get_pfn(pml4_entry), (char *)pdpt);
  size_t pdpt_entry = pdpt[pdpti];
  ptedit_read_physical_page(ptedit_get_pfn(pdpt_entry), (char *)pd);
  size_t pd_entry = pd[pdi];
  ptedit_read_physical_page(ptedit_get_pfn(pd_entry), (char *)pte);

  size_t ptei;
  for (ptei = 0; ptei < 512; ptei++) {
    PTEntry pte_entry;
    pte_entry.entry = pte[ptei];
    pte_entry.vaddr =
        (pml4i << 39) | (pdpti << 30) | (pdi << 21) | (ptei << 12);
    if (!is_present(pte_entry.entry))
      continue;
    entries[ptei] = pte_entry;
  }
  free(pte);
  free(pd);
  free(pml4);
  free(pdpt);
  return entries;
}
