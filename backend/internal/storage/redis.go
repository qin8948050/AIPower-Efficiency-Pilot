package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/qxw/aipower-efficiency-pilot/internal/types"
)

type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisClient(addr string, password string, db int) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password, // no password set
		DB:       db,       // use default DB
	})

	ctx := context.Background()

	// 验证连通性
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

func (r *RedisClient) SavePodTrace(pod *types.PodTrace) error {
	key := fmt.Sprintf("pod_trace:%s:%s", pod.Namespace, pod.PodName)
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}
	// 将 Pod 状态保留 24 小时作为实时快照
	return r.client.Set(r.ctx, key, data, 24*time.Hour).Err()
}

func (r *RedisClient) DeletePodTrace(namespace, podName string) error {
	key := fmt.Sprintf("pod_trace:%s:%s", namespace, podName)
	return r.client.Del(r.ctx, key).Err()
}

func (r *RedisClient) SaveNodePoolID(nodeName string, poolID string) error {
	key := fmt.Sprintf("node_pool:%s", nodeName)
	// 节点池标识较稳定，存活时间长一点
	return r.client.Set(r.ctx, key, poolID, 7*24*time.Hour).Err()
}

func (r *RedisClient) GetNodePoolID(nodeName string) (string, error) {
	key := fmt.Sprintf("node_pool:%s", nodeName)
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", nil // Not found
	}
	return val, err
}
