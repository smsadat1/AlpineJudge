//go:build ignore

// for triggering RE (using Division by zero triggering SIGFPE)
#include <iostream>

int main()
{
    double meh = 55 / 0;
    std::cout << meh;
    return 0;
}