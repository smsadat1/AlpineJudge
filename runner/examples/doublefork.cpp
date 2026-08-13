//go:build ignore
#include <cstdlib>
#include <sys/wait.h>
#include <unistd.h>

int main(int argc, char const *argv[])
{
    pid_t pid = fork();
    // Parent process: Wait for first child and exit quickly
    if(pid > 0) {
        waitpid(pid, NULL, 0);
        return 0;
    } 

    // first child process
    // Second fork to orphan the grandchild
    pid_t grandchild = fork();
    if (grandchild < 0) {
        // second fork failure
        exit(1);
    }

    if (grandchild > 0) {
        // First child exits immediately, reparenting grandchild to PID 1 (init)
        exit(0);
    }

    // grandchild process (Orphaned / Detached Daemon)
    // run forever in background
    for (int i = 0; i < 20; ++i) {
        sleep(1);
    }

    return 0;
}
