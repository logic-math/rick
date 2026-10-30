//go:build darwin

package procfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// New 返回当前平台的进程信息实现（darwin: ps/lsof——macOS 无 procfs）。
// environ 在 darwin 无用户态等价物：返回 ErrUnsupported，调用方降级 cwd/exe 断言。
func New() Info { return darwinInfo{} }

// commandTimeout 限制单次 ps/lsof 调用：这些都跑在收尸/断言的关键路径上，
// 不允许一个卡死的外部命令拖住 release / dev-web。
const commandTimeout = 3 * time.Second

type darwinInfo struct{}

func (darwinInfo) Alive(pid int) bool {
	p, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	if err := p.Signal(syscall.Signal(0)); err != nil {
		// kill(pid,0) POSIX 语义：ESRCH=不存在；EPERM=存在但无权限（如 root
		// 进程）——后者必须视为存在，与 linux 读 /proc/<pid>/stat（全局可读）
		// 的语义对齐；其他错误保守视为存在，再查僵尸态。
		// macOS 差异：对死 pid，Go runtime 把 kill 的 ESRCH 转成 os.ErrProcessDone
		// （"os: process already finished"），errors.Is(ESRCH) 不命中——必须
		// 同检，否则死 pid 被误判「存活」（job_47 跨机后 devweb StopOld 实测）。
		if errors.Is(err, syscall.ESRCH) || errors.Is(err, os.ErrProcessDone) {
			return false
		}
		return !isZombie(pid)
	}
	return !isZombie(pid)
}

// isZombie 用 `ps -o stat=` 查进程状态位（Z 前缀 = 僵尸，与 linux alive()
// 「僵尸态视为已退出」对齐）。ps 失败时返回 false（视为非僵尸 → 保守「存活」：
// 收尸轮询宁可多等，不误判已停）。
func isZombie(pid int) bool {
	out, err := runCommand("ps", "-o", "stat=", "-p", strconv.Itoa(pid))
	if err != nil {
		return false
	}
	return IsZombieState(out)
}

func (darwinInfo) PPID(pid int) (int, error) {
	out, err := runCommand("ps", "-o", "ppid=", "-p", strconv.Itoa(pid))
	if err != nil {
		return 0, fmt.Errorf("ps ppid %d: %w", pid, err)
	}
	v, ok := ParseIntField(out)
	if !ok {
		return 0, fmt.Errorf("ps ppid %d: no parseable output", pid)
	}
	return v, nil
}

func (darwinInfo) Exe(pid int) (string, error) {
	out, err := runCommand("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "txt")
	if err != nil {
		return "", fmt.Errorf("lsof exe %d: %w", pid, err)
	}
	path := ParseLSOFPath(out, "txt")
	if path == "" {
		return "", fmt.Errorf("lsof exe %d: no txt entry", pid)
	}
	return path, nil
}

func (d darwinInfo) ExeHashPath(pid int) (string, error) {
	// darwin 无 /proc 魔术链接：退化为磁盘路径（文件被替换后哈希会变——
	// 指纹主路径是 /api/health 的 build_id，这里只是降级比对，可接受）。
	return d.Exe(pid)
}

func (darwinInfo) Cwd(pid int) (string, error) {
	out, err := runCommand("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd")
	if err != nil {
		return "", fmt.Errorf("lsof cwd %d: %w", pid, err)
	}
	path := ParseLSOFPath(out, "cwd")
	if path == "" {
		return "", fmt.Errorf("lsof cwd %d: no cwd entry", pid)
	}
	return path, nil
}

func (darwinInfo) Environ(int) (map[string]string, error) {
	// darwin 无 /proc/<pid>/environ 等价物（KERN_PROCARGS2 的 env 部分不可靠）。
	// 调用方必须降级到 cwd/exe 断言，不得把 ErrUnsupported 视为「进程不存在」。
	return nil, ErrUnsupported
}

func (darwinInfo) Enumerate() ([]int, error) {
	out, err := runCommand("ps", "-eww", "-o", "pid=")
	if err != nil {
		return nil, fmt.Errorf("ps enumerate: %w", err)
	}
	pids := ParsePIDList(out)
	if len(pids) == 0 {
		return nil, fmt.Errorf("ps enumerate: empty output")
	}
	return pids, nil
}

// runCommand 带超时执行外部命令，返回合并输出；超时与其他错误都返回 err。
func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("%s: timeout after %s", name, commandTimeout)
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// Snapshot 批量采集：1× ps（pid+comm）+ 1× lsof 全量 cwd + 1× lsof 全量 txt。
// 三次子进程调用覆盖全系统（对比逐 pid：数百次 lsof）。
// lsof -F pc 输出形如 "p<pid>\nc<path>"（块状），fd 过滤用 -d cwd / -d txt。
func (darwinInfo) Snapshot() (map[int]ProcSnapshot, error) {
	out := map[int]ProcSnapshot{}
	// ① pid + comm（一次 ps）
	if o, err := runCommand("ps", "-axo", "pid=,comm="); err == nil {
		for _, line := range strings.Split(o, "\n") {
			fields := strings.Fields(strings.TrimSpace(line))
			if len(fields) < 2 {
				continue
			}
			pid, ok := ParseIntField(fields[0])
			if !ok || pid <= 0 {
				continue
			}
			out[pid] = ProcSnapshot{PID: pid, Comm: strings.Join(fields[1:], " ")}
		}
	}
	// ② 全量 cwd（一次 lsof，结构化 -F 输出）
	parseLSOFBlocks := func(o string, field byte) map[int]string {
		res := map[int]string{}
		var curPid int
		for _, line := range strings.Split(o, "\n") {
			if line == "" {
				continue
			}
			if line[0] == 'p' {
				if v, ok := ParseIntField(line[1:]); ok {
					curPid = v
				}
				continue
			}
			if line[0] == field && curPid > 0 {
				res[curPid] = line[1:]
			}
		}
		return res
	}
	if o, err := runCommand("lsof", "-w", "-F", "pc", "-d", "cwd"); err == nil {
		for pid, cwd := range parseLSOFBlocks(o, 'c') {
			s := out[pid]
			s.PID, s.Cwd = pid, cwd
			out[pid] = s
		}
	}
	// ③ 全量 exe（txt 文件描述符）
	if o, err := runCommand("lsof", "-w", "-F", "pn", "-d", "txt"); err == nil {
		for pid, exe := range parseLSOFBlocks(o, 'n') {
			s := out[pid]
			s.PID, s.Exe = pid, exe
			out[pid] = s
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("snapshot: no processes collected")
	}
	return out, nil
}
