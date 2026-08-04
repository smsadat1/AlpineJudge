// bad code to trigger CE
#include <iostream>

int mian() {

    int a, b = 0;
    std::cin >> a >> b;
    std::cout << a + b;

    return 0;   
}