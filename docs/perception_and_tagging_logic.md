# AIPower-Efficiency-Pilot 感知与打标流程详解 (Phase 1)

> **业务目标：** 解决“谁在哪个资源池消耗了多少”的问题，为计费提供权威身份基础。

## 步骤一：资源池身份识别 (Resource Pool Identification)
*   **是什么：** 自动确定 Pod 所驻留的逻辑围栏（如 `Train-H800-Full-Pool`）。
*   **为什么：** 不同池子的 SLA 和采购成本差异巨大，池子身份是计费单价选择的唯一依据。
*   **实现逻辑：** 
    1.  **物理锚定：** `Collector` 捕获 Pod 事件后，立即查询其所在的 `Node` 对象。
    2.  **标签穿透：** 读取 Node 的权威标签 `nvidia.com/pool-id`。
    3.  **兜底策略：** 若 Node 无标签，则根据 GPU 型号匹配默认的同质化池子策略。
*   **效果：** 为 Pod 的生命周期记录打上永久的 `Pool_ID` 标签，确保计费不越权。

## 步骤二：虚拟化模式探测 (Slicing Mode Sensing)
*   **是什么：** 识别 GPU 的底层切割技术（Full, MIG, MPS, TS）。
*   **为什么：** 不同模式代表了不同的资源隔离级别和成本分摊权重。
*   **实现逻辑：** 
    1.  **正则匹配：** 扫描 Pod 资源申请，若存在 `nvidia.com/mig-*` 则判定为 **MIG**。
    2.  **环境变量探测：** 检查 Pod 是否注入了 MPS 共享变量或挂载路径。
    3.  **UUID 关联：** 识别多个 Pod 是否共享同一个物理 `GPU_UUID` 且无硬件隔离（判定为 **TS**）。
*   **效果：** 生成 `Slicing_Mode` 属性，并自动换算计费权重系数。

## 步骤三：实时生命留痕 (Real-time Life-Trace)
*   **是什么：** 建立基于 Kubernetes 权威事件的 Pod 生存时间轴。
*   **为什么：** 解决 Prometheus 指标在任务销毁后标签丢失的问题，提供不可篡改的财务审计存根。
*   **实现逻辑：** 
    1.  **事件监听：** 利用 `client-go` 实时监听 Pod 的启动与删除信号。
    2.  **瞬时持久化：** 将业务身份与起止时间写入 MySQL `life_trace` 表。
*   **效果：** 产生权威的审计存根。

## 测试与验证
1.  **单元测试**: `backend/internal/collector/kubernetes_test.go` (验证 Slicing Mode 和 GPU 提取)。
2.  **集成测试**: `backend/scripts/test_api.sh` (验证 API 返回实时数据)。
3.  **模拟数据**: `backend/scripts/mock_data.go` (清空并重置演示环境)。
