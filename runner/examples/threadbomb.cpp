//go:build ignore
#include <chrono>
#include <thread>
#include <vector>

void worker() {
    // keep thread alive to run out PID
    std::this_thread::sleep_for(std::chrono::seconds(10));
}

// spams POSIX threads
int main(int argc, char const *argv[])
{
    std::vector<std::thread> threads;
    
    while (true)
    {
        // Spawns unjoined threads until PID_LIMIT / memory limit is hit
        threads.emplace_back(worker);
    } 

    return 0;
}
