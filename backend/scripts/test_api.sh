#!/bin/bash

# API 测试脚本

BASE_URL="http://localhost:8080/api/v1"

echo "=== 1. 验证服务健康状况 ==="
curl -s "${BASE_URL}/health" | grep -q "ok" && echo "Health: OK" || echo "Health: FAILED"

echo -e "\n=== 2. 获取资源池统计信息 ==="
POOLS_RESP=$(curl -s "${BASE_URL}/pools")
echo "Pools Response: $POOLS_RESP"
if echo "$POOLS_RESP" | grep -q "id"; then
    echo "Pools Data: OK"
else
    echo "Pools Data: FAILED (No ID found)"
fi

echo -e "\n=== 3. 获取 Pod 追踪列表 ==="
TRACES_RESP=$(curl -s "${BASE_URL}/traces")
echo "Traces Count: $(echo $TRACES_RESP | grep -o "pod_name" | wc -l)"

echo -e "\n=== 测试完成 ==="
