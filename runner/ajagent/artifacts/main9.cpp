#include <iostream>
#include <climits>

int main() {
    volatile int a = INT_MIN;
    volatile int b = -1;

    // Triggers SIGFPE via hardware CPU divide error!
    std::cout << (a / b) << std::endl;

    return 0;
}