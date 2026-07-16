package container

import (
	"fmt"
	"time"
)

// SyncMouseCalibrationResult 一次批量同步的报告.
type SyncMouseCalibrationResult struct {
	Updated []string          `json:"updated"`
	Skipped []string          `json:"skipped"`
	Errors  map[string]string `json:"errors"`
}

// SyncLocalMouseCalibration 遍历 store 里所有本地容器，把主图的 MouseCalibration 节点
// config.counts360 改为 newCounts。
//
// 不变量：
//   - 只改主图 MouseCalibration 节点的 config.counts360
//   - 不动任何节点的 raw dx/dy
//   - Incompatible 容器跳过
func SyncLocalMouseCalibration(store *Store, newCounts int) (SyncMouseCalibrationResult, error) {
	res := SyncMouseCalibrationResult{
		Updated: nil,
		Skipped: nil,
		Errors:  map[string]string{},
	}
	list := store.List()
	for i := range list {
		c := list[i]
		if c.Status == StatusIncompatible {
			continue
		}
		patched := false
		for ni := range c.Graph.Nodes {
			n := &c.Graph.Nodes[ni]
			if n.Kind == "MouseCalibration" {
				SetPinValue(n, "Counts360", newCounts)
				patched = true
			}
		}
		if !patched {
			res.Skipped = append(res.Skipped, c.ID)
			continue
		}
		c.UpdatedAt = time.Now().UTC()
		if err := store.Save(&c); err != nil {
			res.Errors[c.ID] = err.Error()
			continue
		}
		res.Updated = append(res.Updated, c.ID)
	}
	if len(res.Errors) == 0 {
		return res, nil
	}
	return res, fmt.Errorf("部分容器同步失败：%v", res.Errors)
}
