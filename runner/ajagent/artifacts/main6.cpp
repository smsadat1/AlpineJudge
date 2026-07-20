#include <unistd.h>

// segfault

int main(int argc, char const *argv[])
{
    int *p = NULL;
    p[1] = 69;
    return 0;
}
