#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>
#include <sys/wait.h>
#include <unistd.h>

#define NOP16                                                                  \
  asm volatile("nop\nnop\nnop\nnop\nnop\nnop\nnop\nnop\nnop\nnop\nnop\nnop\nn" \
               "op\nnop\nnop\nnop\n");
#define NOP256                                                                 \
  NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16 NOP16      \
      NOP16 NOP16 NOP16 NOP16
#define NOP4K                                                                  \
  NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 NOP256 \
      NOP256 NOP256 NOP256 NOP256 NOP256

#define COLOR_RED "\x1b[31m"
#define COLOR_GREEN "\x1b[32m"
#define COLOR_YELLOW "\x1b[33m"
#define COLOR_RESET "\x1b[0m"

#define TAG_OK COLOR_GREEN "[+]" COLOR_RESET " "
#define TAG_FAIL COLOR_RED "[-]" COLOR_RESET " "
#define TAG_PROGRESS COLOR_YELLOW "[~]" COLOR_RESET " "

void nx_function() {
  NOP4K
  printf("Hello\n");
  NOP4K
}

#if defined(__i386__) || defined(__x86_64__)
#define NX_BIT PTEDIT_PAGE_BIT_NX
#elif defined(__aarch64__)
#define NX_BIT PTEDIT_PAGE_BIT_XN
#endif

int main(int argc, char *argv[]) {
  pid_t pid = getpid();
  printf("PID: %d\n", pid);

  /* Get 4kb-aligned pointer to function */
  void *nx_function_aligned = (void *)((((size_t)nx_function) + 4096) & ~0xfff);

  printf(TAG_PROGRESS "Expect 'Hello': ");
  nx_function();

  /* Make function non-executable (calling it now leads to crash) */
  if (mprotect(nx_function_aligned, 4096, PROT_READ)) {
    exit(42);
  }
  printf("nx_function2: %p\n", nx_function_aligned);
  char input[11];
  printf("write something...\n");
  fgets(input, sizeof(input), stdin);

  nx_function();
}
