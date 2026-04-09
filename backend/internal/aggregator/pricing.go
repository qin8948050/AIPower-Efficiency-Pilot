package aggregator

import (
	"fmt"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/storage"
)

// SlicingWeightOf 返回指定切片模式对应的计费权重
func SlicingWeightOf(pricing storage.PoolPricing, slicingMode string) float64 {
	switch slicingMode {
	case "MIG":
		return pricing.SlicingWeightMIG
	case "MPS":
		return pricing.SlicingWeightMPS
	case "TS":
		return pricing.SlicingWeightTS
	default: // Full
		return pricing.SlicingWeightFull
	}
}

// CalculateCost 根据会话时长、池子单价与切片权重计算成本
// cost = durationHours × basePricePerHour × slicingWeight
// slicingWeight 由调用方（stitcher）根据 Pod 申请的切片单元数和池子最大切片单元数计算
func CalculateCost(startTime, endTime time.Time, pricing storage.PoolPricing, slicingWeight float64) float64 {
	durationH := endTime.Sub(startTime).Hours()
	if durationH < 0 {
		durationH = 0
	}
	return durationH * pricing.BasePricePerHour * slicingWeight
}

// DefaultPricing 返回一个预置的默认定价配置（用于无真实 DB 数据时的 Fallback）
func DefaultPricing(poolID string) storage.PoolPricing {
	priceMap := map[string]float64{
		"pool-v100-shared":    28.0,
		"pool-a100-priority":  55.0,
		"pool-t4-lowcost":     12.0,
	}
	price, ok := priceMap[poolID]
	if !ok {
		price = 30.0
	}
	return storage.PoolPricing{
		PoolID:            poolID,
		GPUModel:          "Auto",
		BasePricePerHour:  price,
		SlicingWeightFull: 1.0,
		SlicingWeightMIG:  0.35,
		SlicingWeightMPS:  0.5,
		SlicingWeightTS:   0.6,
	}
}

// GetPoolPricing 从 MySQL 加载定价配置，若不存在则返回默认值
func GetPoolPricing(db *storage.MySQLClient, poolID string) storage.PoolPricing {
	p, err := db.GetPoolPricing(poolID)
	if err != nil || p == nil {
		fmt.Printf("[pricing] pool '%s' 未配置定价，使用默认值\n", poolID)
		return DefaultPricing(poolID)
	}
	return *p
}
