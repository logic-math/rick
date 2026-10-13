package env

import (
	"os"
	"os/user"
	"strconv"
)

// RealHomeDir 返回**真实**家目录（不跟随 $HOME）：dev 实例与 release 提升链的
// HOME 可能已被切换（dev HOME），但种子文件、构建缓存与生产 state 都在真实
// 家目录下。优先 os/user（linux: /etc/passwd；darwin: 系统目录服务，CGO=0
// 也可用），失败回退 $HOME；两者都无 → 空串（调用方自行报错）。
func RealHomeDir() string {
	if u, err := user.LookupId(strconv.Itoa(os.Getuid())); err == nil && u.HomeDir != "" {
		return u.HomeDir
	}
	return os.Getenv("HOME")
}
