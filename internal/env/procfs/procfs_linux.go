//go:build linux

package procfs

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// New 返回当前平台的进程信息实现（linux: /proc，逐条保留原 devweb/release
// 私有实现的语义——包括「stat 不可读→not alive」「malformed→保守 alive」）。
func New() Info { return linuxInfo{} }

type linuxInfo struct{}

// readStatAfterComm 读取 /proc/<pid>/stat 的 comm 之后的字段串（state 起始）。
// readable=false 表示文件不可读（进程不存在或无权限）；malformed=true 表示
// 可读但 comm 边界解析失败。
func readStatAfterComm(pid int) (after string, readable, malformed bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return "", false, false
	}
	// stat: pid (comm) state ... —— comm 可能含空格/括号，取最后一个 ')'
	idx := strings.LastIndexByte(string(data), ')')
	if idx < 0 || idx+2 >= len(data) {
		return "", true, true
	}
	return string(data)[idx+2:], true, false
}

func (linuxInfo) Alive(pid int) bool {
	after, readable, malformed := readStatAfterComm(pid)
	if !readable {
		return false // 进程不存在（或读不到）→ 与原实现一致
	}
	if malformed {
		return true // 解析不了 → 保守视为存活（原实现同）
	}
	return after[0] != 'Z' // 僵尸态视为已退出
}

func (linuxInfo) PPID(pid int) (int, error) {
	after, readable, malformed := readStatAfterComm(pid)
	if !readable || malformed {
		return 0, fmt.Errorf("read /proc/%d/stat", pid)
	}
	fields := strings.Fields(after)
	if len(fields) < 2 {
		return 0, fmt.Errorf("malformed /proc/%d/stat fields", pid)
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, err
	}
	return ppid, nil
}

func (linuxInfo) Exe(pid int) (string, error) {
	exe, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(exe, " (deleted)"), nil
}

func (linuxInfo) ExeHashPath(pid int) (string, error) {
	// /proc/<pid>/exe 魔术链接：磁盘文件被替换后仍指向旧 inode，
	// 哈希它 = 哈希运行中的真实二进制。
	return fmt.Sprintf("/proc/%d/exe", pid), nil
}

func (linuxInfo) Cwd(pid int) (string, error) {
	return os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
}

func (linuxInfo) Environ(pid int) (map[string]string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, kv := range strings.Split(string(data), "\x00") {
		if kv == "" {
			continue
		}
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue // 异常条目（无 '='）跳过；调用方只做精确键值断言
		}
		out[k] = v
	}
	return out, nil
}

func (linuxInfo) Command(pid int) (string, error) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", err
	}
	cmd := strings.ReplaceAll(strings.TrimSuffix(string(data), "\x00"), "\x00", " ")
	if cmd == "" {
		return "", fmt.Errorf("/proc/%d/cmdline: empty", pid)
	}
	return cmd, nil
}

func (linuxInfo) Enumerate() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid <= 0 {
			continue
		}
		out = append(out, pid)
	}
	return out, nil
}

// Snapshot 在 linux 返回 ErrUnsupported：linux 的逐 pid /proc 读取是 µs 级，
// 批量快照无收益（它是为 darwin 的逐 pid lsof 分钟级痛点设计的）；
// 调用方（findProdPIDs/findDevPIDs）对本错误回退到原逐 pid 路径。
func (linuxInfo) Snapshot() (map[int]ProcSnapshot, error) {
	return nil, ErrUnsupported
}
