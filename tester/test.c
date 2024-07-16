#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int main() {
  pid_t pid = getpid();
  printf("This process has ID: %ld\n", (long)pid);

  unsigned int hello = 0;
  printf("hello (virtual address): %p\n", (void *)&hello);

  // Add some more stack variables
  unsigned int var1 = 1;
  unsigned int var2 = 2;
  unsigned int var3 = 3;
  printf("var1 (virtual address): %p\n", (void *)&var1);
  printf("var2 (virtual address): %p\n", (void *)&var2);
  printf("var3 (virtual address): %p\n", (void *)&var3);

  // Allocate a large buffer to consume more heap space
  void *buffer = malloc(1024 * 1024 * 10); // 10 MB buffer
  if (buffer == NULL) {
    perror("malloc");
    return 1;
  }
  printf("buffer (virtual address): %p\n", (void *)buffer);

  // Allocate the memory you are interested in
  void *i = malloc(20 * sizeof(size_t));
  if (i == NULL) {
    perror("malloc");
    free(buffer);
    return 1;
  }
  printf("i (virtual address): %p\n", (void *)i);
  free(i);

  // Clean up the large buffer
  free(buffer);

  char input[11];
  fgets(input, sizeof(input), stdin);
  return 0;
}
