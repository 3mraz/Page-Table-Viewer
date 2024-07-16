#include <memory.h>
#include <stdio.h>
#include <string.h>
#include <sys/mman.h>
#include <unistd.h>

int main() {
  pid_t pid = getpid();
  printf("This process has ID: %ld\n", (long)pid);

  char *target = mmap(0, 4096, PROT_READ, MAP_PRIVATE | MAP_ANONYMOUS, -1, 0);
  printf("target (virtual address): %p\n", target);

  printf("%s\n", target);
  char input[11];
  fgets(input, sizeof(input), stdin);

  strcpy(target, input);
  printf("%s\n", target);
  if (munmap(target, 4096) == -1) {
    perror("munmap");
    return 1;
  }
  return 0;
}
