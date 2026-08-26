package store

import (
	"os"
	"path/filepath"
	"runtime/debug"
	"testing"
)

func countOpenFDs(t *testing.T) int {
	t.Helper()
	d, err := os.Open("/dev/fd")
	if err != nil {
		t.Skipf("无法读取 /dev/fd: %v", err)
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		t.Fatalf("读取 /dev/fd 失败: %v", err)
	}
	return len(names)
}

// TestPersistWriteClosesFileDescriptors 连续写入快照不得累积文件句柄。
func TestPersistWriteClosesFileDescriptors(t *testing.T) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)

	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatal(err)
	}
	base := countOpenFDs(t)
	snap := &snapshot{Version: snapshotVersion, Seq: map[string]int64{}}
	for i := 0; i < 150; i++ {
		if err := p.Write(snap); err != nil {
			t.Fatalf("写入快照失败: %v", err)
		}
	}
	after := countOpenFDs(t)
	if after-base > 50 {
		t.Fatalf("Write 泄漏文件句柄: %d -> %d", base, after)
	}
}

// TestPersistSyncDirClosesDescriptor 同步目录不得泄漏目录句柄。
func TestPersistSyncDirClosesDescriptor(t *testing.T) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)

	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatal(err)
	}
	base := countOpenFDs(t)
	for i := 0; i < 150; i++ {
		if err := p.syncDir(); err != nil {
			t.Fatalf("同步目录失败: %v", err)
		}
	}
	after := countOpenFDs(t)
	if after-base > 50 {
		t.Fatalf("syncDir 泄漏文件句柄: %d -> %d", base, after)
	}
}

// TestPersistReadClosesDescriptor 反复加载快照不得累积文件句柄。
func TestPersistReadClosesDescriptor(t *testing.T) {
	old := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(old)

	dir := t.TempDir()
	p, err := NewJSONPersister(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Write(&snapshot{Version: snapshotVersion, Seq: map[string]int64{}}); err != nil {
		t.Fatal(err)
	}
	base := countOpenFDs(t)
	for i := 0; i < 100; i++ {
		if _, err := New(Options{DataDir: dir}); err != nil {
			t.Fatalf("打开仓储失败: %v", err)
		}
	}
	after := countOpenFDs(t)
	if after-base > 50 {
		t.Fatalf("Read 泄漏文件句柄: %d -> %d", base, after)
	}
}

// TestStoreClosePropagatesSaveError 落盘失败时 Close 必须返回错误，不能吞掉。
func TestStoreClosePropagatesSaveError(t *testing.T) {
	dir := t.TempDir()
	st, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	// 把 store.json 替换成同名目录，使原子替换必然失败（root 下 chmod 不生效）。
	snapPath := filepath.Join(dir, snapshotFile)
	if err := os.Remove(snapPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapPath, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := st.Close(); err == nil {
		t.Fatalf("落盘失败时 Close 应返回错误，实际被吞掉")
	}
}
