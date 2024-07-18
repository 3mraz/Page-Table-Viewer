#include <errno.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "../ptedit_header.h"
#define NX_BIT PTEDIT_PAGE_BIT_NX

int main(int argc, char *argv[]) {
  if (ptedit_init()) {
    fprintf(stderr, "Error initializing ptedit\n");
    return 1;
  }

  char pidstr[20];
  printf("Enter pid: ");
  if (fgets(pidstr, sizeof(pidstr), stdin) == NULL) {
    fprintf(stderr, "Error reading PID\n");
    ptedit_cleanup();
    return 1;
  }
  int pid = atoi(pidstr);
  if (pid <= 0) {
    fprintf(stderr, "Invalid PID\n");
    ptedit_cleanup();
    return 1;
  }
  printf("The pid is: %d\n", pid);

  char hex_string[100];
  printf("Enter virt. address: ");
  if (scanf("%99s", hex_string) != 1) {
    fprintf(stderr, "Error reading virtual address\n");
    ptedit_cleanup();
    return 1;
  }

  // Remove "0x" prefix if present
  if (strncmp(hex_string, "0x", 2) == 0 || strncmp(hex_string, "0X", 2) == 0) {
    memmove(hex_string, hex_string + 2, strlen(hex_string) - 1);
  }

  char *endptr;
  void *virt_addr = (void *)strtoull(hex_string, &endptr, 16);

  if (*endptr != '\0') {
    fprintf(stderr, "Invalid hex string\n");
    ptedit_cleanup();
    return 1;
  }

  printf("Virtual address: %p\n", virt_addr);

  ptedit_pte_clear_bit(virt_addr, pid, PTEDIT_PAGE_BIT_RW);
  ptedit_cleanup();
  return 0;
}
