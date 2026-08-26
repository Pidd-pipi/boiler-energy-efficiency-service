package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStore_CorruptSnapshotDegradesGracefully 验证损坏快照会被备份为 .bak，
// 并以空库启动而不是返回错误。
func TestStore_CorruptSnapshotDegradesGracefully(t *testing.T) {
	dir := t.TempDir()
	snapPath := filepath.Join(dir, snapshotFile)
	badContent := []byte("{ not valid json")
	if err := os.WriteFile(snapPath, badContent, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := New(Options{DataDir: dir})
	if err != nil {
		t.Fatalf("损坏快照应降级启动而非报错: %v", err)
	}
	if s.CountBoilers() != 0 {
		t.Fatalf("降级后应为空库，实际锅炉数: %d", s.CountBoilers())
	}
	// 原路径已重建为合法空快照，不再保留损坏内容。
	cur, err := os.ReadFile(snapPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cur), "{ not valid json") {
		t.Fatalf("原路径不应继续保留损坏内容")
	}
	// 坏文件已被备份为 .bak。
	bak, err := os.ReadFile(snapPath + ".bak")
	if err != nil {
		t.Fatalf("应生成 .bak 备份: %v", err)
	}
	if string(bak) != string(badContent) {
		t.Fatalf(".bak 内容应与损坏原文一致")
	}
}
