#include <memory.h>
#include <stdio.h>
#include <stdlib.h>
#include <sys/mman.h>
#include <unistd.h>

int main() {
  pid_t pid = getpid();
  printf("PID: %d\n", pid);

  char *buffer =
      mmap(0, 4096, PROT_READ | PROT_WRITE, MAP_ANONYMOUS | MAP_PRIVATE, -1, 0);
  memset(buffer, 0x42, 4096);
  printf("buffer (virtual address): %p\n", (void *)buffer);

  char input[11];
  printf("Write something... \n");
  fgets(input, sizeof(input), stdin);

  char *evict = (char *)malloc(4096 * 2048);
  for (int i = 0; i < 2048; i++) {
    evict[i * 4096] = (i % 2) + 1;
  }

  strcpy(buffer, input);
  printf("Buffer content: %s\n", buffer);
  munmap(buffer, 4096);
  return 0;
}
