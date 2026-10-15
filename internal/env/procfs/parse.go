package procfs

import (
	"strconv"
	"strings"
)

// ParsePIDList 解析 `ps -o pid=` 风格的多行 pid 输出：逐行取数字字段，
// 忽略空行/非数字行/非正数（pid 0 不存在）。
func ParsePIDList(out string) []int {
	var pids []int
	for _, line := range strings.Split(out, "\n") {
		if v, ok := ParseIntField(line); ok {
			pids = append(pids, v)
		}
	}
	return pids
}

// ParseIntField 解析 ps 单值输出（如 `ps -o ppid= -p <pid>` 的一行 " 123"），
// 多字段时取末尾字段；非正数视为不可解析。
func ParseIntField(out string) (int, bool) {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return 0, false
	}
	v, err := strconv.Atoi(fields[len(fields)-1])
	if err != nil || v <= 0 {
		return 0, false
	}
	return v, true
}

// IsZombieState 判断 `ps -o stat=` 输出是否僵尸态（Z 前缀，如 "Z"、"Z+"）。
// 无法解析（空/异常）→ false（视为非僵尸：收尸轮询宁可多等，不误判已停）。
func IsZombieState(stat string) bool {
	return strings.HasPrefix(strings.TrimSpace(stat), "Z")
}

// lsofTypeKinds 是 lsof TYPE 列的常见取值（用于锚定 FD 列：FD 之后紧跟 TYPE）。
var lsofTypeKinds = map[string]bool{
	"REG": true, "DIR": true, "CHR": true, "BLK": true, "FIFO": true,
	"UNIX": true, "IPv4": true, "IPv6": true, "SOCK": true, "PIPE": true,
	"DEL": true, "LINK": true, "VREG": true, "VDIR": true,
}

// ParseLSOFPath 从 lsof 输出中取 FD 列等于 fd 的首行 NAME 列。
// lsof 列布局：COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME（9 列）。
// COMMAND 与 NAME 都可能含空格，无法从左或右硬锚定 → 结构化扫描：找值等于
// fd 且其后紧跟 TYPE 列取值（REG/DIR/...）、前方至少容得下 COMMAND/PID/USER
// （i≥3）、后方至少容得下 TYPE/DEVICE/SIZE-OFF/NODE + NAME≥1（i+5≤len-1）
// 的字段。NAME 为其后第 5 个字段起全部合并（连续空格会被折叠）。无匹配返回 ""。
// 残留歧义：路径里恰好出现「txt REG」这样的相邻组件会被误判（未见实际场景）。
func ParseLSOFPath(out, fd string) string {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		for i := 3; i+5 <= len(fields)-1; i++ {
			if fields[i] != fd || !lsofTypeKinds[fields[i+1]] {
				continue
			}
			return strings.Join(fields[i+5:], " ")
		}
	}
	return ""
}
