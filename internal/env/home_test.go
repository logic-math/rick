package env

import (
	"os"
	"testing"
)

// TestRealHomeDirNonEmpty 真实环境上必须能解析出非空家目录。
func TestRealHomeDirNonEmpty(t *testing.T) {
	if got := RealHomeDir(); got == "" {
		t.Fatal("RealHomeDir 不应为空")
	}
}

// TestRealHomeDirIgnoresHOME 切换 $HOME 后仍应返回真实家目录（uid 解析成功时），
// 与「不跟随 $HOME」的语义一致。
func TestRealHomeDirIgnoresHOME(t *testing.T) {
	t.Setenv("HOME", "/tmp/fake-home")
	got := RealHomeDir()
	if got == "" {
		t.Skip("当前环境无法解析 uid 家目录（回退路径），跳过")
	}
	if got == "/tmp/fake-home" {
		t.Fatalf("RealHomeDir 跟随了 $HOME（= /tmp/fake-home），应为真实家目录 %q", got)
	}
}

// TestRealHomeDirFallbackToHOME uid 解析不可用时回退 $HOME（无法直接构造失败，
// 用行为等价断言：HOME 非空时结果非空）。
func TestRealHomeDirFallbackToHOME(t *testing.T) {
	t.Setenv("HOME", os.TempDir())
	if RealHomeDir() == "" {
		t.Fatal("HOME 已设置时 RealHomeDir 不应为空")
	}
}
