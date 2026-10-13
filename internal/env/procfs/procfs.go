// Package procfs 提供跨平台的进程信息抽象（linux: /proc；darwin: ps/lsof），
// 供 devweb / release 的进程发现、所有权断言与安全收尸使用。
//
// 设计动机：原实现直接读 /proc，在 macOS 上（无 procfs）全部静默失效——
// alive() 恒 false 导致「进程已死」误判、pid 文件被删、端口被旧进程占用。
// 本包把「需要什么信息」与「从哪里取」解耦：linux 实现逐条保留原 /proc 语义，
// darwin 实现用 ps/lsof 等价替换（environ 除外：darwin 无 /proc/<pid>/environ
// 等价物，返回 ErrUnsupported，调用方降级到 cwd/exe 断言，不得视为「进程不存在」）。
package procfs

import "errors"

// ErrUnsupported 表示当前平台不支持该查询。调用方应把它与其他错误区分：
// 降级到替代断言（cwd/exe），而不是直接判失败或误判进程不存在。
var ErrUnsupported = errors.New("procfs: unsupported on this platform")

// Info 是进程信息查询抽象。
type Info interface {
	// Alive 报告 pid 是否仍存在；僵尸态视为已退出（收尸轮询语义）。
	// 进程不存在或信息不可读时返回 false（与原 /proc 实现一致：
	// 宁可误判已死〔跳过候选〕，也不误判存活）。
	Alive(pid int) bool

	// PPID 返回父进程 pid（祖先链遍历用）。
	PPID(pid int) (int, error)

	// Exe 返回进程可执行文件的解析路径（linux: readlink /proc/<pid>/exe，
	// 去除 " (deleted)" 后缀；darwin: lsof txt 行）。用于所有权断言的前缀匹配。
	Exe(pid int) (string, error)

	// ExeHashPath 返回「内容即运行中二进制」的路径：linux 是 /proc/<pid>/exe
	// 魔术链接（磁盘文件被替换后仍指向旧 inode，保住指纹比对语义）；darwin
	// 无等价物，退化为磁盘路径（Exe 同源）——指纹主路径本就是 /api/health 的
	// build_id，这里只是降级比对。
	ExeHashPath(pid int) (string, error)

	// Cwd 返回进程工作目录。
	Cwd(pid int) (string, error)

	// Environ 返回进程环境变量。darwin 返回 ErrUnsupported（无等价物）。
	Environ(pid int) (map[string]string, error)

	// Enumerate 枚举系统全部 pid（>0；按平台自然顺序，不排序——
	// 与原 os.ReadDir("/proc") 的名字序保持一致）。getpid 的过滤在调用方。
	Enumerate() ([]int, error)
}

// unsupportedInfo 是未知平台的降级实现：所有查询失败但不 panic
// （alive=false / 其余 ErrUnsupported），让上层安全拒杀而不是崩溃。
type unsupportedInfo struct{}

func (unsupportedInfo) Alive(int) bool                         { return false }
func (unsupportedInfo) PPID(int) (int, error)                  { return 0, ErrUnsupported }
func (unsupportedInfo) Exe(int) (string, error)                { return "", ErrUnsupported }
func (unsupportedInfo) ExeHashPath(int) (string, error)        { return "", ErrUnsupported }
func (unsupportedInfo) Cwd(int) (string, error)                { return "", ErrUnsupported }
func (unsupportedInfo) Environ(int) (map[string]string, error) { return nil, ErrUnsupported }
func (unsupportedInfo) Enumerate() ([]int, error)              { return nil, ErrUnsupported }
