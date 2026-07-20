#include <cassert>

// trigger abort

int main(int argc, char const *argv[])
{
    int x = 9;
    assert(x == 10);
    return 0;
}
