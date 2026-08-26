package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"example.com/boiler-energy-efficiency-service/domain"
)

// snapshotFile 持久化文件名。
const snapshotFile = "store.json"

// snapshotVersion 快照格式版本。
const snapshotVersion = 1

// snapshot 是落盘的完整数据快照。
type snapshot struct {
	Version    int                        `json:"version"`
	SavedAt    time.Time                  `json:"saved_at"`
	Boilers    []*domain.Boiler           `json:"boilers"`
	RunData    []*domain.RunData          `json:"run_data"`
	Efficiency []*domain.EfficiencyRecord `json:"efficiency"`
	Combustion []*domain.CombustionStatus `json:"combustion"`
	Alerts     []*domain.RunAlert         `json:"alerts"`
	Blowdown   []*domain.BlowdownRecord   `json:"blowdown"`
	Reports    []*domain.DailyReport      `json:"reports"`
	Audit      []*domain.AuditEntry       `json:"audit"`
	Seq        map[string]int64           `json:"seq"`
}

// CorruptSnapshotError 表示快照文件损坏或版本不兼容。
type CorruptSnapshotError struct {
	Path string
	Err  error
}

func (e *CorruptSnapshotError) Error() string {
	return fmt.Sprintf("快照文件损坏 %s: %v", e.Path, e.Err)
}

func (e *CorruptSnapshotError) Unwrap() error { return e.Err }

// JSONPersister 负责把内存数据写入 JSON 文件（原子替换）。
type JSONPersister struct {
	dir string
}

// NewJSONPersister 创建持久化器并确保数据目录存在。
func NewJSONPersister(dir string) (*JSONPersister, error) {
	if dir == "" {
		return nil, errors.New("数据目录不能为空")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	return &JSONPersister{dir: dir}, nil
}

// Path 返回快照文件完整路径。
func (p *JSONPersister) Path() string {
	return filepath.Join(p.dir, snapshotFile)
}

// BackupPath 返回损坏快照的备份路径。
func (p *JSONPersister) BackupPath() string {
	return p.Path() + ".bak"
}

// Write 原子写入快照：写临时文件 -> fsync -> rename -> fsync 目录。
func (p *JSONPersister) Write(snap *snapshot) (err error) {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化快照失败: %w", err)
	}
	tmp := p.Path() + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("打开快照临时文件失败: %w", err)
	}
	// 统一收尾：成功路径也要关闭临时文件句柄，否则每次落盘都泄漏一个 fd，
	// 批量写入跑久就会触发 "too many open files"。
	defer func() {
		if cerr := f.Close(); err == nil && cerr != nil {
			err = fmt.Errorf("关闭快照临时文件失败: %w", cerr)
		}
		// 任何错误路径下都清理临时文件，避免残留。
		if err != nil {
			_ = os.Remove(tmp)
		}
	}()
	if _, err = f.Write(data); err != nil {
		return fmt.Errorf("写入快照临时文件失败: %w", err)
	}
	if err = f.Sync(); err != nil {
		return fmt.Errorf("同步快照临时文件失败: %w", err)
	}
	if err = os.Rename(tmp, p.Path()); err != nil {
		return fmt.Errorf("替换快照文件失败: %w", err)
	}
	// fsync 目录，保证 rename 结果在掉电后仍可见。
	if err = p.syncDir(); err != nil {
		return fmt.Errorf("同步快照目录失败: %w", err)
	}
	return nil
}

// syncDir 同步数据目录元数据，保证 rename 结果在掉电后仍可见。
func (p *JSONPersister) syncDir() error {
	d, err := os.Open(p.dir)
	if err != nil {
		return err
	}
	defer d.Close()
	if err := d.Sync(); err != nil {
		return err
	}
	return nil
}

// BackupCorrupt 将损坏的快照备份为 .bak，避免后续启动反复失败。
func (p *JSONPersister) BackupCorrupt() error {
	src := p.Path()
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	dst := p.BackupPath()
	if _, err := os.Stat(dst); err == nil {
		dst = fmt.Sprintf("%s.%d", dst, time.Now().UnixNano())
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("备份损坏快照失败: %w", err)
	}
	return nil
}

// Read 读取快照；文件不存在时返回 (nil, nil)。
// 文件损坏或版本不兼容时返回 *CorruptSnapshotError。
func (p *JSONPersister) Read() (*snapshot, error) {
	f, err := os.Open(p.Path())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("读取快照失败: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("读取快照失败: %w", err)
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, &CorruptSnapshotError{Path: p.Path(), Err: fmt.Errorf("解析快照失败: %w", err)}
	}
	if snap.Version != snapshotVersion {
		return nil, &CorruptSnapshotError{Path: p.Path(), Err: fmt.Errorf("快照版本不兼容: %d != %d", snap.Version, snapshotVersion)}
	}
	return &snap, nil
}

// snapshotLocked 在持锁状态下构造快照副本。
func (s *Store) snapshotLocked() *snapshot {
	snap := &snapshot{
		Version: snapshotVersion,
		SavedAt: s.now(),
		Seq:     make(map[string]int64, len(s.seq)),
	}
	for _, id := range s.boilerOrder {
		snap.Boilers = append(snap.Boilers, s.boilers[id])
	}
	for _, id := range s.runDataOrder {
		snap.RunData = append(snap.RunData, s.runData[id])
	}
	for _, id := range s.efficiencyOrder {
		snap.Efficiency = append(snap.Efficiency, s.efficiency[id])
	}
	for _, id := range s.combustionOrder {
		snap.Combustion = append(snap.Combustion, s.combustion[id])
	}
	for _, id := range s.alertOrder {
		snap.Alerts = append(snap.Alerts, s.alerts[id])
	}
	for _, id := range s.blowdownOrder {
		snap.Blowdown = append(snap.Blowdown, s.blowdown[id])
	}
	for _, id := range s.reportOrder {
		snap.Reports = append(snap.Reports, s.reports[id])
	}
	snap.Audit = append(snap.Audit, s.audit...)
	for k, v := range s.seq {
		snap.Seq[k] = v
	}
	return snap
}

// loadFromDisk 从磁盘恢复数据并重建索引。
func (s *Store) loadFromDisk() error {
	snap, err := s.persister.Read()
	if err != nil {
		return err
	}
	if snap == nil {
		// 首次启动：落一个空快照，保证目录可写。
		return s.persister.Write(&snapshot{
			Version: snapshotVersion,
			SavedAt: s.now(),
			Seq:     make(map[string]int64),
		})
	}
	for k, v := range snap.Seq {
		s.seq[k] = v
	}
	for _, b := range snap.Boilers {
		s.boilers[b.ID] = b
		s.boilerOrder = append(s.boilerOrder, b.ID)
	}
	for _, rd := range snap.RunData {
		s.runData[rd.ID] = rd
		s.runDataOrder = append(s.runDataOrder, rd.ID)
		s.runDataByBoiler[rd.BoilerID] = append(s.runDataByBoiler[rd.BoilerID], rd.ID)
	}
	for _, e := range snap.Efficiency {
		s.efficiency[e.ID] = e
		s.efficiencyOrder = append(s.efficiencyOrder, e.ID)
		s.efficiencyByBoiler[e.BoilerID] = append(s.efficiencyByBoiler[e.BoilerID], e.ID)
	}
	for _, c := range snap.Combustion {
		s.combustion[c.ID] = c
		s.combustionOrder = append(s.combustionOrder, c.ID)
		s.combustionByBoiler[c.BoilerID] = append(s.combustionByBoiler[c.BoilerID], c.ID)
	}
	for _, a := range snap.Alerts {
		s.alerts[a.ID] = a
		s.alertOrder = append(s.alertOrder, a.ID)
		s.alertsByBoiler[a.BoilerID] = append(s.alertsByBoiler[a.BoilerID], a.ID)
	}
	for _, b := range snap.Blowdown {
		s.blowdown[b.ID] = b
		s.blowdownOrder = append(s.blowdownOrder, b.ID)
		s.blowdownByBoiler[b.BoilerID] = append(s.blowdownByBoiler[b.BoilerID], b.ID)
	}
	for _, r := range snap.Reports {
		key := reportKey(r.BoilerID, r.Date)
		s.reports[key] = r
		s.reportOrder = append(s.reportOrder, key)
	}
	s.audit = append(s.audit, snap.Audit...)
	return nil
}

// handleLoadError 对损坏快照降级为空库启动；其它错误原样返回。
func (s *Store) handleLoadError(err error) error {
	var corrupt *CorruptSnapshotError
	if !errors.As(err, &corrupt) {
		return err
	}
	slog.Warn("快照文件损坏，将备份后降级为空库启动", "path", corrupt.Path, "err", corrupt.Err)
	if berr := s.persister.BackupCorrupt(); berr != nil {
		slog.Error("备份损坏快照失败", "err", berr)
	} else {
		slog.Warn("已备份损坏快照", "backup", s.persister.BackupPath())
	}
	if werr := s.persister.Write(&snapshot{
		Version: snapshotVersion,
		SavedAt: s.now(),
		Seq:     make(map[string]int64),
	}); werr != nil {
		slog.Error("降级后写入空快照失败", "err", werr)
	}
	return nil
}
