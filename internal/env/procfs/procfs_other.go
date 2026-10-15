//go:build !linux && !darwin

package procfs

// New 未知平台：降级实现（所有查询失败但不 panic），上层安全拒杀。
// linux/darwin 的实现在各自构建标签文件里。
func New() Info { return unsupportedInfo{} }
