#define _GNU_SOURCE

#include <errno.h>
#include <stdio.h>
#include <string.h>
#include <sys/mount.h>
#include <sys/ptrace.h>
#include <sys/reboot.h>
#include <sys/syscall.h>
#include <unistd.h>

struct TestResult {
    int ret;
    int err;
};

struct Test {
    const char *name;
    struct TestResult (*run)(void);
};

/* ----------------- Individual Tests ----------------- */

static struct TestResult test_mount(void)
{
    errno = 0;
    int ret = mount("tmpfs", "/tmp/nonexistent", "tmpfs", 0, "");
    return (struct TestResult){
        .ret = ret, .err = errno
    };
}

static struct TestResult test_umount(void)
{
    errno = 0;
    int ret = umount("/");
    return (struct TestResult){
        .ret = ret, .err = errno
    };
}

static struct TestResult test_chroot(void)
{
    errno = 0;
    int ret = chroot("/");
    return (struct TestResult){
        .ret = ret, .err = errno
    };
}

static struct TestResult test_pivot_root(void)
{
    errno = 0;
    int ret = syscall(SYS_pivot_root, "/", "/");
    return (struct TestResult){
        .ret = ret, .err = errno
    };
}

static struct TestResult test_ptrace(void)
{
    errno = 0;
    int ret = ptrace(PTRACE_TRACEME, 0, NULL, NULL);
    return (struct TestResult){
        .ret = ret, .err = errno
    };
}

static struct TestResult test_reboot(void)
{
    errno = 0;
    int ret = reboot(RB_AUTOBOOT);
    return (struct TestResult) {
        .ret = ret, .err = errno
    };
};


/* ----------------- Test Table ----------------- */

static struct Test tests[] = {
    { "mount",      test_mount      },
    { "umount",     test_umount     },
    { "chroot",     test_chroot     },
    { "pivot_root", test_pivot_root },
    { "ptrace",     test_ptrace     },
    { "reboot",     test_reboot     },
};

int main(void)
{
    size_t total = sizeof(tests) / sizeof(tests[0]);

    puts("== Sandbox Privileged Syscall Test ==\n");

    for (size_t i = 0; i < total; i++) {
        struct TestResult r = tests[i].run();

        if (r.ret == -1) {
            printf("[DENIED] %-12s errno=%d (%s)\n",
                   tests[i].name, r.err, strerror(r.err));
        } else {
            printf("[ALLOWED] %-12s\n", tests[i].name);
        }
    }

    return 0;
}