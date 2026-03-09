package aggregator

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// RunDailyAggregation 执行日级聚合，将前一天的 life_trace 聚合写入 daily_billing_snapshot
func RunDailyAggregation(db *storage.MySQLClient, date time.Time) error {
	snapshotDate := date.Format("2006-01-02")
	log.Printf("[daily] 开始执行 %s 的日级账单聚合...", snapshotDate)

	// 查询当天所有已结束且已缝合指标的记录
	sessions, err := db.GetBillingSessions(snapshotDate, "", "", "")
	if err != nil {
		return fmt.Errorf("查询日账单记录失败: %w", err)
	}
	if len(sessions) == 0 {
		log.Printf("[daily] %s 无账单记录，跳过", snapshotDate)
		return nil
	}

	// 按 poolID + namespace 分组聚合
	type groupKey struct {
		PoolID    string
		Namespace string
		TeamLabel string
	}
	type groupVal struct {
		TotalCost int64 // 存储 cost * 10000 以保持精度
		UtilVals  []float64
		MaxMemGiB float64
		Count     int
	}

	groups := make(map[groupKey]*groupVal)
	for _, s := range sessions {
		k := groupKey{PoolID: s.PoolID, Namespace: s.Namespace, TeamLabel: s.TeamLabel}
		if groups[k] == nil {
			groups[k] = &groupVal{}
		}
		g := groups[k]
		g.TotalCost += int64(s.CostAmount * 10000)
		g.UtilVals = append(g.UtilVals, s.GPUUtilAvg)
		memGiB := float64(s.MemUsedMax) / 1024.0
		if memGiB > g.MaxMemGiB {
			g.MaxMemGiB = memGiB
		}
		g.Count++
	}

	// 写入 daily_billing_snapshot
	for k, g := range groups {
		p95 := calcP95(g.UtilVals)
		snapshot := &storage.DailyBillingSnapshot{
			SnapshotDate:    snapshotDate,
			PoolID:          k.PoolID,
			Namespace:       k.Namespace,
			TeamLabel:       k.TeamLabel,
			TotalCost:       float64(g.TotalCost) / 10000,
			AvgUtilP95:      p95,
			MaxMemGiB:       g.MaxMemGiB,
			PodSessionCount: g.Count,
		}
		if err := db.UpsertDailySnapshot(snapshot); err != nil {
			log.Printf("[daily] 写入快照失败 %v: %v", k, err)
			continue
		}
		log.Printf("[daily] ✅ %s/%s totalCost=¥%.4f p95=%.2f%% sessions=%d",
			k.Namespace, k.PoolID, snapshot.TotalCost, p95, g.Count)
	}
	log.Printf("[daily] %s 日级聚合完成，共 %d 个分组", snapshotDate, len(groups))
	return nil
}

func calcP95(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted))*0.95+0.5) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
