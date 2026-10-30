//go:build darwin

package procfs

import (
	"testing"
	"time"
)

func TestSnapshotBatchSpeedAndContent(t *testing.T) {
	t0 := time.Now()
	snaps, err := New().Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	elapsed := time.Since(t0)
	if elapsed > 10*time.Second {
		t.Fatalf("Snapshot 太慢: %v（应为 1-3 次子进程调用的量级）", elapsed)
	}
	if len(snaps) < 5 {
		t.Fatalf("进程数异常少: %d", len(snaps))
	}
	withCwd, withExe := 0, 0
	for _, s := range snaps {
		if s.Cwd != "" {
			withCwd++
		}
		if s.Exe != "" {
			withExe++
		}
	}
	t.Logf("processes=%d withCwd=%d withExe=%d elapsed=%v", len(snaps), withCwd, withExe, elapsed)
	if withCwd == 0 && withExe == 0 {
		t.Fatal("cwd/exe 全空——批量解析坏了")
	}
}
