package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/qxw/aipower-efficiency-pilot/internal/types"
)

type MySQLClient struct {
	db *sql.DB
}

func NewMySQLClient(dsn string) (*MySQLClient, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql: %v", err)
	}

	// 设置连接池
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping mysql: %v", err)
	}

	client := &MySQLClient{db: db}
	if err := client.InitSchema(); err != nil {
		return nil, fmt.Errorf("failed to init mysql schema: %v", err)
	}

	return client, nil
}

func (m *MySQLClient) InitSchema() error {
	query := `
	CREATE TABLE IF NOT EXISTS life_trace (
		id INT AUTO_INCREMENT PRIMARY KEY,
		pod_uid VARCHAR(64) NOT NULL,
		namespace VARCHAR(64) NOT NULL,
		pod_name VARCHAR(128) NOT NULL,
		node_name VARCHAR(128) NOT NULL,
		pool_id VARCHAR(128) NOT NULL,
		slicing_mode VARCHAR(32) NOT NULL,
		start_time DATETIME NOT NULL,
		end_time DATETIME,
		INDEX idx_pod_uid (pod_uid),
		INDEX idx_time (start_time)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *MySQLClient) SaveLifeTrace(trace *types.PodTrace) error {
	query := `
	INSERT INTO life_trace (pod_uid, namespace, pod_name, node_name, pool_id, slicing_mode, start_time)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON DUPLICATE KEY UPDATE 
		node_name=VALUES(node_name), 
		pool_id=VALUES(pool_id), 
		slicing_mode=VALUES(slicing_mode);
	`
	_, err := m.db.Exec(query,
		trace.PodUID,
		trace.Namespace,
		trace.PodName,
		trace.NodeName,
		trace.PoolID,
		string(trace.SlicingMode),
		trace.StartTime)
	return err
}

func (m *MySQLClient) CloseLifeTrace(namespace, podName string) error {
	query := `
	UPDATE life_trace SET end_time = ? 
	WHERE namespace = ? AND pod_name = ? AND end_time IS NULL 
	ORDER BY start_time DESC LIMIT 1;
	`
	_, err := m.db.Exec(query, time.Now(), namespace, podName)
	return err
}

func (m *MySQLClient) GetActivePodTrace(namespace, podName string) (*types.PodTrace, error) {
	// 这是一个辅助方法，方便后续对齐数据
	query := `
	SELECT pod_uid, namespace, pod_name, node_name, pool_id, slicing_mode, start_time 
	FROM life_trace 
	WHERE namespace = ? AND pod_name = ? AND end_time IS NULL 
	ORDER BY start_time DESC LIMIT 1;
	`
	var t types.PodTrace
	var slicingMode string
	err := m.db.QueryRow(query, namespace, podName).Scan(
		&t.PodUID, &t.Namespace, &t.PodName, &t.NodeName, &t.PoolID, &slicingMode, &t.StartTime,
	)
	if err != nil {
		return nil, err
	}
	t.SlicingMode = types.SlicingMode(slicingMode)
	return &t, nil
}
