#include <vector>
#include <cstring>
#include <iostream>

int main() {

    const size_t CHUNK_SIZE = 10 * 1024 * 1024; // 10 MB    
    std::vector<char*> chunks;
    
    while (true) {
        char* ptr = new char[CHUNK_SIZE];
        
        // memset forces the Linux kernel to allocate physical RAM pages (RSS)
        std::memset(ptr, 1, CHUNK_SIZE);
        chunks.push_back(ptr);
    }

    return 0;
}