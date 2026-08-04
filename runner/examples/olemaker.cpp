// spam log buffer flush to trigger OLE
#include <iostream>

int main(int argc, char const *argv[])
{
    for (int i = 0; i < 1000000; i++)
    {
        std::cout << "LOL " << std::endl;
    }
    return 0;
}