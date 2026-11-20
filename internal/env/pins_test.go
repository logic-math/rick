package env

import (
	"reflect"
	"testing"
)

// ---------------------------------------------------------------------------
// 版本固定表（job_55：pi + pi 扩展锁版本）
// ---------------------------------------------------------------------------

// TestPinnedExtensionSpec 覆盖固定 spec 生成：已知包返回带版本 spec，
// 未知包返回空串。
func TestPinnedExtensionSpec(t *testing.T) {
	cases := []struct {
		pkg  string
		want string
	}{
		{"pi-subagents", "npm:pi-subagents@0.51.0"},
		{"pi-web-access", "npm:pi-web-access@0.24.0"},
		{"", ""},
		{"unknown-pkg", ""},
		{"pi-subagents-extra", ""},
	}
	for _, tc := range cases {
		if got := PinnedExtensionSpec(tc.pkg); got != tc.want {
			t.Errorf("PinnedExtensionSpec(%q) = %q, want %q", tc.pkg, got, tc.want)
		}
	}
}

// TestPinnedExtensionVersion 覆盖固定版本查询：已知包返回版本号，未知包返回空串。
func TestPinnedExtensionVersion(t *testing.T) {
	cases := []struct {
		pkg  string
		want string
	}{
		{"pi-subagents", "0.51.0"},
		{"pi-web-access", "0.24.0"},
		{"", ""},
		{"unknown-pkg", ""},
	}
	for _, tc := range cases {
		if got := PinnedExtensionVersion(tc.pkg); got != tc.want {
			t.Errorf("PinnedExtensionVersion(%q) = %q, want %q", tc.pkg, got, tc.want)
		}
	}
}

// TestPackageNameFromSource 覆盖注册源名 → 裸包名提取（npm: 前缀与 @version
// 后缀剥离，兼容 scoped 包名）。
func TestPackageNameFromSource(t *testing.T) {
	cases := []struct {
		source string
		want   string
	}{
		{"npm:pi-subagents@0.51.0", "pi-subagents"},
		{"npm:pi-subagents", "pi-subagents"},
		{"pi-subagents", "pi-subagents"},
		{"npm:@scope/pkg@1.0.0", "@scope/pkg"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := packageNameFromSource(tc.source); got != tc.want {
			t.Errorf("packageNameFromSource(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}

// TestRequiredExtensionsMatchesPins 验证 RequiredExtensions 与固定表派生一致
// （单一来源：pin 表同时拥有名字与版本）。
func TestRequiredExtensionsMatchesPins(t *testing.T) {
	if got, want := RequiredExtensions, pinnedExtensionNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("RequiredExtensions = %v, want %v (derived from PinnedExtensions)", got, want)
	}
	if len(RequiredExtensions) == 0 {
		t.Error("RequiredExtensions must not be empty")
	}
}
