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
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (LifeTrace) TableName() string {
	return "life_trace"
}

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

func (m *MySQLClient) InitSchema() error {
	return m.db.AutoMigrate(&LifeTrace{})
}

// TruncateTable 清空生命留痕表 (仅用于测试/Mock)
func (m *MySQLClient) TruncateTable() error {
	return m.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&LifeTrace{}).Error
}

func (m *MySQLClient) SaveLifeTrace(trace *types.PodTrace) error {
	lt := &LifeTrace{
		PodUID:      trace.PodUID,
		Namespace:   trace.Namespace,
		PodName:     trace.PodName,
		NodeName:    trace.NodeName,
		PoolID:      trace.PoolID,
		SlicingMode: string(trace.SlicingMode),
		StartTime:   trace.StartTime,
	}

	// 使用 Upsert 逻辑 (基于 PodUID)
	return m.db.Where(LifeTrace{PodUID: lt.PodUID}).
		Assign(LifeTrace{
			NodeName:    lt.NodeName,
			PoolID:      lt.PoolID,
			SlicingMode: lt.SlicingMode,
		}).
		FirstOrCreate(lt).Error
}

func (m *MySQLClient) CloseLifeTrace(namespace, podName string) error {
	now := time.Now()
	return m.db.Model(&LifeTrace{}).
		Where("namespace = ? AND pod_name = ? AND end_time IS NULL", namespace, podName).
		Order("start_time DESC").
		Limit(1).
		Update("end_time", &now).Error
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
