//go:build linux

package procfs

import (
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// TestLinuxInfoAgainstRealProcess 用真实子进程验证 linux 实现的六个查询。
func TestLinuxInfoAgainstRealProcess(t *testing.T) {
	info := New()

	// ① 本进程存活 + 关键字段
	if !info.Alive(os.Getpid()) {
		t.Fatal("Alive(自己) 应为 true")
	}
	if info.Alive(999999) {
		t.Fatal("Alive(不存在的 pid) 应为 false")
	}
	if ppid, err := info.PPID(os.Getpid()); err != nil || ppid != os.Getppid() {
		t.Fatalf("PPID = (%d, %v)，期望 (%d, nil)", ppid, err, os.Getppid())
	}
	if env, err := info.Environ(os.Getpid()); err != nil || len(env) == 0 {
		t.Fatalf("Environ(自己) = (%d 项, %v)，期望非空且无错", len(env), err)
	}
	if pids, err := info.Enumerate(); err != nil || len(pids) == 0 {
		t.Fatalf("Enumerate = (%d 个, %v)，期望非空且无错", len(pids), err)
	} else {
		found := false
		for _, p := range pids {
			if p == os.Getpid() {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("Enumerate 应包含本进程")
		}
	}

	// ② spawn 一个带已知环境的子进程，验证 Environ 断言面与 Cwd/Exe
	sleep, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("no sleep binary: %v", err)
	}
	dir := t.TempDir()
	cmd := exec.Command(sleep, "30")
	cmd.Env = []string{"HOME=/fake/home", "RICK_STATE_DIR=/fake/state", "PATH=" + os.Getenv("PATH")}
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})
	pid := cmd.Process.Pid

	// 等 /proc/<pid>/environ 就绪（spawn 后立即可读，但保守轮询）
	deadline := time.Now().Add(2 * time.Second)
	var env map[string]string
	for time.Now().Before(deadline) {
		if env, err = info.Environ(pid); err == nil && env["HOME"] == "/fake/home" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || env["HOME"] != "/fake/home" || env["RICK_STATE_DIR"] != "/fake/state" {
		t.Fatalf("Environ(子进程) 未读到已知环境: (%d 项, %v)", len(env), err)
	}
	if cwd, err := info.Cwd(pid); err != nil || cwd != dir {
		t.Fatalf("Cwd(子进程) = (%q, %v)，期望 %q", cwd, err, dir)
	}
	exe, err := info.Exe(pid)
	if err != nil || exe != sleep {
		t.Fatalf("Exe(子进程) = (%q, %v)，期望 %q", exe, err, sleep)
	}
	hashPath, err := info.ExeHashPath(pid)
	if err != nil || hashPath != "/proc/"+strconv.Itoa(pid)+"/exe" {
		t.Fatalf("ExeHashPath(子进程) = (%q, %v)", hashPath, err)
	}
	if _, err := os.Stat(hashPath); err != nil {
		t.Fatalf("ExeHashPath 应可读: %v", err)
	}
	if ppid, err := info.PPID(pid); err != nil || ppid != os.Getpid() {
		t.Fatalf("PPID(子进程) = (%d, %v)，期望本进程 %d", ppid, err, os.Getpid())
	}
	if !info.Alive(pid) {
		t.Fatal("Alive(子进程) 应为 true")
	}

	// ③ 杀掉子进程后 Alive 转 false
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill sleep: %v", err)
	}
	_, _ = cmd.Process.Wait()
	if info.Alive(pid) {
		t.Fatal("Wait 之后的子进程 Alive 应为 false")
	}
}
