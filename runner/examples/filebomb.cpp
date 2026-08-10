// go:build ignore
#include <fstream>
#include <iostream>
#include <string>
#include <vector>


// file descriptor spam
int main(int argc, char const *argv[])
{
    std::vector<std::ofstream> files;

    for (int i = 0; i < 2147483646; i++) {
        std::string filename = "file_" + std::to_string(i) + ".txt";
        files.emplace_back(filename);

        // Check if file failed to open (indicates EMFILE / ENFILE)
        if(!files.back().is_open()) {
            return 1;
        }
    }
    return 0;
}
