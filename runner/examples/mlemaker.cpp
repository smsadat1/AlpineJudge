//go:build ignore

// to trigger MLE by abusing memory
#include <vector>

int main(int argc, char const *argv[])
{
    std::vector<int> mem;
    while (true)
    {
        mem.push_back(420);
    }
    
    return 0;
}
