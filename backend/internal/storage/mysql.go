package storage

import (
	"fmt"
	"time"

	"github.com/qxw/aipower-efficiency-pilot/internal/types"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// LifeTrace 数据库模型
type LifeTrace struct {
	ID          uint       `gorm:"primaryKey"`
	PodUID      string     `gorm:"column:pod_uid;type:varchar(64);not null;index:idx_pod_uid"`
	Namespace   string     `gorm:"column:namespace;type:varchar(64);not null"`
	PodName     string     `gorm:"column:pod_name;type:varchar(128);not null"`
	NodeName    string     `gorm:"column:node_name;type:varchar(128);not null"`
	PoolID      string     `gorm:"column:pool_id;type:varchar(128);not null"`
	SlicingMode string     `gorm:"column:slicing_mode;type:varchar(32);not null"`
	StartTime   time.Time  `gorm:"column:start_time;type:datetime;not null;index:idx_time"`
	EndTime     *time.Time `gorm:"column:end_time;type:datetime"`
	// 状态: Running, Auditing, Settled
	Status      string     `gorm:"column:status;type:varchar(32);default:'Running'"`
	// 业务归属标签（从 Pod Labels 提取）
	TeamLabel    string `gorm:"column:team_label;type:varchar(128)"`
	ProjectLabel string `gorm:"column:project_label;type:varchar(128)"`
	// Pod 声明需要的硬件特性（从 Pod Annotations 提取）
	// 如: "NVLink,RDMA,FP8"
	RequiredFeatures string `gorm:"column:required_features;type:varchar(256)"`
	// 聚合指标持久化字段（Phase 2 指标缝合后回填）
	GPUUtilAvg    float64 `gorm:"column:gpu_util_avg;type:float;default:0"`
	GPUUtilMax    float64 `gorm:"column:gpu_util_max;type:float;default:0"`
	MemUsedMax    uint64  `gorm:"column:mem_used_max;type:bigint;default:0"`
	PowerUsageAvg float64 `gorm:"column:power_usage_avg;type:float;default:0"`
	CostAmount    float64 `gorm:"column:cost_amount;type:decimal(12,4);default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (LifeTrace) TableName() string {
	return "life_trace"
}

// PoolPricing 资源池定价配置
type PoolPricing struct {
	ID                 uint    `gorm:"primaryKey"`
	PoolID             string  `gorm:"column:pool_id;type:varchar(128);not null;uniqueIndex"`
	GPUModel           string  `gorm:"column:gpu_model;type:varchar(64);not null"`
	BasePricePerHour   float64 `gorm:"column:base_price_per_hour;type:decimal(10,4);not null"`
	SlicingWeightFull  float64 `gorm:"column:slicing_weight_full;type:float;default:1.0"`
	SlicingWeightMIG   float64 `gorm:"column:slicing_weight_mig;type:float;default:0.35"`
	SlicingWeightMPS   float64 `gorm:"column:slicing_weight_mps;type:float;default:0.5"`
	SlicingWeightTS    float64 `gorm:"column:slicing_weight_ts;type:float;default:0.6"`
}

func (PoolPricing) TableName() string { return "pool_pricing" }

// DailyBillingSnapshot 日级账单聚合快照
type DailyBillingSnapshot struct {
	ID              uint      `gorm:"primaryKey"`
	SnapshotDate    string    `gorm:"column:snapshot_date;type:date;not null;index:idx_snapshot"`
	PoolID          string    `gorm:"column:pool_id;type:varchar(128);not null;index:idx_snapshot"`
	Namespace       string    `gorm:"column:namespace;type:varchar(64);not null;index:idx_snapshot"`
	TeamLabel       string    `gorm:"column:team_label;type:varchar(128)"`
	TotalCost       float64   `gorm:"column:total_cost;type:decimal(12,4);default:0"`
	AvgUtilP95      float64   `gorm:"column:avg_util_p95;type:float;default:0"`
	MaxMemGiB       float64   `gorm:"column:max_mem_gib;type:float;default:0"`
	PodSessionCount int       `gorm:"column:pod_session_count;type:int;default:0"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (DailyBillingSnapshot) TableName() string { return "daily_billing_snapshot" }

type MySQLClient struct {
	db *gorm.DB
}

func NewMySQLClient(dsn string) (*MySQLClient, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect mysql: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	client := &MySQLClient{db: db}
	if err := client.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to init schema: %v", err)
	}

	return client, nil
}

// ResourcePool 资源池元数据 (基于 K8s 标签感知的逻辑资产)
type ResourcePool struct {
	ID               string    `gorm:"column:pool_id;type:varchar(128);primaryKey"`
	Name             string    `gorm:"column:name;type:varchar(128);not null"`
	Scene            string    `gorm:"column:scene;type:varchar(64)"` // 预训练、推理、研发等
	GPUModel         string    `gorm:"column:gpu_model;type:varchar(64)"`
	HardwareFeatures string    `gorm:"column:hardware_features;type:varchar(255)"` // NVLink, RDMA 等
	SlicingMode      string    `gorm:"column:slicing_mode;type:varchar(32)"`      // Full, MIG, MPS, TS
	PricingLogic     string    `gorm:"column:pricing_logic;type:varchar(64)"`     // Reserved, Spot 等
	Priority         string    `gorm:"column:priority;type:varchar(32)"`          // High, Low
	Description      string    `gorm:"column:description;type:text"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (ResourcePool) TableName() string { return "resource_pool" }

func (m *MySQLClient) InitSchema() error {
	return m.db.AutoMigrate(&LifeTrace{}, &PoolPricing{}, &DailyBillingSnapshot{}, &ResourcePool{}, &InsightReport{})
}

// UpsertResourcePool 发现新池子时自动创建或更新硬指标 (型号、特性、模式)
func (m *MySQLClient) UpsertResourcePool(p *ResourcePool) error {
	return m.db.Where(ResourcePool{ID: p.ID}).
		Assign(ResourcePool{
			GPUModel:         p.GPUModel,
			HardwareFeatures: p.HardwareFeatures,
			SlicingMode:      p.SlicingMode,
		}).
		FirstOrCreate(p).Error
}

// UpdateResourcePoolMetadata 手动更新业务层面的元数据 (别名、场景、定价、优先级、备注)
func (m *MySQLClient) UpdateResourcePoolMetadata(p *ResourcePool) error {
	return m.db.Model(&ResourcePool{}).Where("pool_id = ?", p.ID).
		Updates(map[string]interface{}{
			"name":          p.Name,
			"scene":         p.Scene,
			"pricing_logic": p.PricingLogic,
			"priority":      p.Priority,
			"description":   p.Description,
		}).Error
}

// GetAllResourcePools 获取所有已感知的资源池
func (m *MySQLClient) GetAllResourcePools() ([]ResourcePool, error) {
	var pools []ResourcePool
	err := m.db.Order("priority desc, pool_id asc").Find(&pools).Error
	return pools, err
}

// TruncateTable 清空生命留痕表 (仅用于测试/Mock)
func (m *MySQLClient) TruncateTable() error {
	return m.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&LifeTrace{}).Error
}

// GetPendingMetricsTraces 查询已结束但尚未缝合指标的 LifeTrace 记录
func (m *MySQLClient) GetPendingMetricsTraces(limit int) ([]LifeTrace, error) {
	var traces []LifeTrace
	err := m.db.Where("end_time IS NOT NULL AND gpu_util_avg = 0").
		Order("end_time DESC").
		Limit(limit).
		Find(&traces).Error
	return traces, err
}

// UpdateLifeTraceMetrics 将离线缝合后的指标与费用持久化到 MySQL
func (m *MySQLClient) UpdateLifeTraceMetrics(id uint, avgUtil, maxUtil float64, maxMem uint64, avgPower, cost float64) error {
	return m.db.Model(&LifeTrace{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"gpu_util_avg":    avgUtil,
			"gpu_util_max":    maxUtil,
			"mem_used_max":    maxMem,
			"power_usage_avg": avgPower,
			"cost_amount":     cost,
			"status":          "Settled",
		}).Error
}

// GetPoolPricing 查询指定资源池的定价配置
func (m *MySQLClient) GetPoolPricing(poolID string) (*PoolPricing, error) {
	var pricing PoolPricing
	err := m.db.Where("pool_id = ?", poolID).First(&pricing).Error
	if err != nil {
		return nil, err
	}
	return &pricing, nil
}

// SavePoolPricing 创建或更新资源池定价配置
func (m *MySQLClient) SavePoolPricing(pricing *PoolPricing) error {
	return m.db.Where(PoolPricing{PoolID: pricing.PoolID}).
		Assign(*pricing).
		FirstOrCreate(pricing).Error
}

func (m *MySQLClient) SaveLifeTrace(trace *types.PodTrace) error {
	lt := &LifeTrace{
		PodUID:       trace.PodUID,
		Namespace:    trace.Namespace,
		PodName:      trace.PodName,
		NodeName:     trace.NodeName,
		PoolID:       trace.PoolID,
		SlicingMode:  string(trace.SlicingMode),
		StartTime:    trace.StartTime,
		Status:       "Running",
		TeamLabel:    trace.TeamLabel,
		ProjectLabel: trace.ProjectLabel,
	}

	// 使用 Upsert 逻辑 (基于 PodUID)
	return m.db.Where(LifeTrace{PodUID: lt.PodUID}).
		Assign(LifeTrace{
			NodeName:     lt.NodeName,
			PoolID:       lt.PoolID,
			SlicingMode:  lt.SlicingMode,
			TeamLabel:    lt.TeamLabel,
			ProjectLabel: lt.ProjectLabel,
		}).
		FirstOrCreate(lt).Error
}

func (m *MySQLClient) CloseLifeTrace(namespace, podName string) error {
	now := time.Now()
	return m.db.Model(&LifeTrace{}).
		Where("namespace = ? AND pod_name = ? AND end_time IS NULL", namespace, podName).
		Order("start_time DESC").
		Limit(1).
		Updates(map[string]interface{}{
			"end_time": &now,
			"status":   "Auditing",
		}).Error
}

// GetBillingSessions 查询指定日期的账单记录 (已结束的 Pod)
func (m *MySQLClient) GetBillingSessions(dateStr, poolID, namespace, teamLabel string) ([]LifeTrace, error) {
	startTime, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %v", err)
	}
	endTime := startTime.Add(24 * time.Hour)

	query := m.db.Model(&LifeTrace{}).
		Where("end_time >= ? AND end_time < ?", startTime, endTime)

	if poolID != "" {
		query = query.Where("pool_id = ?", poolID)
	}
	if namespace != "" {
		query = query.Where("namespace = ?", namespace)
	}
	if teamLabel != "" {
		query = query.Where("team_label = ?", teamLabel)
	}

	var traces []LifeTrace
	err = query.Order("end_time DESC").Find(&traces).Error
	return traces, err
}

// UpsertDailySnapshot 插入或更新日级账单快照
func (m *MySQLClient) UpsertDailySnapshot(snapshot *DailyBillingSnapshot) error {
	// 根据唯一索引 (snapshot_date, pool_id, namespace) 进行 Upsert
	return m.db.Where(DailyBillingSnapshot{
		SnapshotDate: snapshot.SnapshotDate,
		PoolID:       snapshot.PoolID,
		Namespace:    snapshot.Namespace,
	}).Assign(DailyBillingSnapshot{
		TeamLabel:       snapshot.TeamLabel,
		TotalCost:       snapshot.TotalCost,
		AvgUtilP95:      snapshot.AvgUtilP95,
		MaxMemGiB:       snapshot.MaxMemGiB,
		PodSessionCount: snapshot.PodSessionCount,
	}).FirstOrCreate(snapshot).Error
}

// GetDailySnapshots 查询日级账单快照
func (m *MySQLClient) GetDailySnapshots(dateStr string) ([]DailyBillingSnapshot, error) {
	var snapshots []DailyBillingSnapshot
	query := m.db.Model(&DailyBillingSnapshot{})
	if dateStr != "" {
		query = query.Where("snapshot_date = ?", dateStr)
	}
	err := query.Order("snapshot_date DESC, total_cost DESC").Find(&snapshots).Error
	return snapshots, err
}

// GetAllPoolPricing 获取所有资源池定价配置
func (m *MySQLClient) GetAllPoolPricing() ([]PoolPricing, error) {
	var pricing []PoolPricing
	err := m.db.Find(&pricing).Error
	return pricing, err
}

// RawExec 执行原始 SQL (仅用于演示/Mock)
func (m *MySQLClient) RawExec(sql string) error {
	return m.db.Exec(sql).Error
}

// SaveRawLifeTrace 直接保存完整的生命留痕记录 (包含聚合指标)
func (m *MySQLClient) SaveRawLifeTrace(lt *LifeTrace) error {
	return m.db.Create(lt).Error
}

// GetDailySnapshotsByPool 查询指定资源池的日级账单快照（用于 LLM 摘要）
func (m *MySQLClient) GetDailySnapshotsByPool(poolID string, startDate, endDate time.Time) ([]DailyBillingSnapshot, error) {
	var snapshots []DailyBillingSnapshot
	err := m.db.Where("pool_id = ? AND snapshot_date >= ? AND snapshot_date < ?", poolID, startDate.Format("2006-01-02"), endDate.Format("2006-01-02")).
		Order("snapshot_date DESC").
		Find(&snapshots).Error
	return snapshots, err
}

// GetPodTracesByPool 查询指定资源池的 Pod 记录
func (m *MySQLClient) GetPodTracesByPool(poolID string, startDate, endDate time.Time) ([]LifeTrace, error) {
	var traces []LifeTrace
	err := m.db.Where("pool_id = ? AND start_time >= ? AND start_time < ?", poolID, startDate, endDate).
		Order("start_time DESC").
		Find(&traces).Error
	return traces, err
}

// GetPodTracesByPoolAndUtil 查询指定资源池中利用率低于阈值的 Pod
func (m *MySQLClient) GetPodTracesByPoolAndUtil(poolID string, startDate, endDate time.Time, maxUtil float64) ([]LifeTrace, error) {
	var traces []LifeTrace
	err := m.db.Where("pool_id = ? AND start_time >= ? AND start_time < ? AND gpu_util_avg < ?", poolID, startDate, endDate, maxUtil).
		Order("gpu_util_avg ASC").
		Find(&traces).Error
	return traces, err
}

// GetResourcePool 获取单个资源池信息
func (m *MySQLClient) GetResourcePool(poolID string) (*ResourcePool, error) {
	var pool ResourcePool
	err := m.db.Where("pool_id = ?", poolID).First(&pool).Error
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

// InsightReport AI 诊断报告模型
// 核心对象是任务（Pod/PyTorchJob），而非资源池
type InsightReport struct {
	ID          uint      `gorm:"primaryKey"`
	GeneratedAt time.Time `gorm:"column:generated_at"`
	TaskName    string    `gorm:"column:task_name;index"`    // 任务名
	Namespace   string    `gorm:"column:namespace"`           // 命名空间
	Team        string    `gorm:"column:team"`                // 负责团队
	PoolID      string    `gorm:"column:pool_id;index"`      // 当前所在资源池
	ReportType  string    `gorm:"column:report_type"`
	Summary     string    `gorm:"column:summary;type:text"`
	RootCause   string    `gorm:"column:root_cause;type:text"`
	Actions     string    `gorm:"column:actions;type:text"`
	EstSavings  float64   `gorm:"column:est_savings"`
	Status      string    `gorm:"column:status;default:'pending'"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (InsightReport) TableName() string {
	return "insight_reports"
}

// SaveInsightReport 保存诊断报告
func (m *MySQLClient) SaveInsightReport(report *InsightReport) error {
	return m.db.Create(report).Error
}

// GetInsightReports 获取诊断报告列表
func (m *MySQLClient) GetInsightReports(poolID string, limit, offset int) ([]InsightReport, int64, error) {
	var reports []InsightReport
	var total int64

	query := m.db.Model(&InsightReport{})
	if poolID != "" {
		query = query.Where("pool_id = ?", poolID)
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	err = query.Order("generated_at DESC").Limit(limit).Offset(offset).Find(&reports).Error
	return reports, total, err
}

// GetInsightReportByID 获取单个报告
func (m *MySQLClient) GetInsightReportByID(id uint) (*InsightReport, error) {
	var report InsightReport
	err := m.db.Where("id = ?", id).First(&report).Error
	if err != nil {
		return nil, err
	}
	return &report, nil
}

// UpdateInsightReportStatus 更新报告状态
func (m *MySQLClient) UpdateInsightReportStatus(id uint, status string) error {
	return m.db.Model(&InsightReport{}).Where("id = ?", id).Update("status", status).Error
}

func (m *MySQLClient) GetActivePodTrace(namespace, podName string) (*types.PodTrace, error) {
	var lt LifeTrace
	err := m.db.Where("namespace = ? AND pod_name = ? AND end_time IS NULL", namespace, podName).
		Order("start_time DESC").
		First(&lt).Error
	if err != nil {
		return nil, err
	}

	return &types.PodTrace{
		PodUID:      lt.PodUID,
		Namespace:   lt.Namespace,
		PodName:     lt.PodName,
		NodeName:    lt.NodeName,
		PoolID:      lt.PoolID,
		SlicingMode: types.SlicingMode(lt.SlicingMode),
		StartTime:   lt.StartTime,
	}, nil
}
