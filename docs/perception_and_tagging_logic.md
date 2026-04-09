# AIPower-Efficiency-Pilot 感知与打标流程详解 (Phase 1)

> **业务目标：** 解决”谁在哪个资源池消耗了多少”的问题，为计费提供权威身份基础。

## 步骤一：资源池身份识别 (Resource Pool Identification)
*   **是什么：** 自动确定 Pod 所驻留的逻辑围栏（如 `pool-full-A100`, `pool-mig-A100-2g`）。
*   **为什么：** 不同池子的 SLA 和采购成本差异巨大，池子身份是计费单价选择的唯一依据。
*   **实现逻辑：**
    1.  **物理锚定：** `Collector` 捕获 Pod 事件后，立即查询其所在的 `Node` 对象。
    2.  **标签穿透：** 读取 Node 的权威标签 `nvidia.com/pool-id`。
    3.  **池子命名规范：** 格式 `pool-{mode}-{vendor}-{spec}`，如：
        - `pool-full-A100` → Full 模式，A100，权重 1.0
        - `pool-mig-A100-2g` → MIG 2g 规格，A100，MIG 最大 7 实例
        - `pool-ts-T4-30` → TS 模式，T4，30% 切片
    4.  **兜底策略：** 若 Node 无标签，则根据 GPU 型号匹配默认的同质化池子策略。
*   **效果：** 为 Pod 的生命周期记录打上永久的 `Pool_ID` 标签，确保计费不越权。

## 步骤二：切片配置与权重计算 (Slicing Config & Weight)
*   **是什么：** 基于资源池配置和 Pod 申请量，动态计算切片权重。
*   **为什么：** 权重应该由 Pod 实际申请的切片单元数计算，而非固定值，确保计费公平。
*   **数据模型 (`resource_pool` 表)：**

    | 字段 | 类型 | 说明 |
    |---|---|---|
    | `pool_id` | VARCHAR | 资源池 ID |
    | `gpu_model` | VARCHAR | GPU 型号（如 `NVIDIA A100-SXM4-80GB`）|
    | `gpu_vendor` | VARCHAR | 厂商（nvidia, intel, amd）|
    | `slicing_mode` | VARCHAR | 切片模式：Full, MIG, MPS, TS |
    | `max_slicing_units` | INT | 单卡最大切片单元数（如 A100 MIG=7）|

*   **实现逻辑：**
    1.  **节点配置解析：** 从 `nvidia.com/mig.config` 标签或池子名称推断切片模式和最大单元数。
    2.  **Pod 申请量统计：** `countSlicingUnits()` 统计 Pod 申请的切片数：
        - **Full**: 请求 `nvidia.com/gpu=N` → 返回 N
        - **MIG**: 统计 `nvidia.com/mig-*` 资源请求总和
        - **MPS**: 从 `CUDA_MPS_ACTIVE_THREAD_PERCENTAGE` 环境变量获取百分比
        - **TS**: 从 `nvidia.com/gpu-percentage` 注解获取百分比
    3.  **权重计算：**
        ```
        SlicingWeight = Pod申请的切片单元数 / 池子的最大切片单元数

        例如:
        - Pod 申请 2 个 MIG 实例，池子 max=7 → Weight = 2/7 ≈ 0.286
        - Pod 设置 MPS 50%，池子 max=100 → Weight = 50/100 = 0.5
        ```

*   **效果：** 生成 `SlicingMode`、`SlicingUnits`、`SlicingWeight` 属性，实现精确到 Pod 级别的动态计费。

## 步骤三：业务归属提取 (Business Attribution)
*   **是什么：** 从 Pod Labels 提取业务归属信息。
*   **实现逻辑：**
    1.  从 `app.kubernetes.io/team` 提取团队标签
    2.  从 `app.kubernetes.io/project` 提取项目标签
*   **效果：** 支持按团队/项目的成本分摊下钻。

## 步骤四：实时生命留痕 (Real-time Life-Trace)
*   **是什么：** 建立基于 Kubernetes 权威事件的 Pod 生存时间轴。
*   **为什么：** 解决 Prometheus 指标在任务销毁后标签丢失的问题，提供不可篡改的财务审计存根。
*   **实现逻辑：**
    1.  **事件监听：** 利用 `client-go` 实时监听 Pod 的启动与删除信号。
    2.  **瞬时持久化：** 将业务身份、切片配置与起止时间写入 MySQL `life_trace` 表。
    3.  **结束时间：** 使用 Pod 的 `DeletionTimestamp` 而非 Collector 收到事件的时间，确保精确。
*   **效果：** 产生权威的审计存根，包含精确的 SlicingWeight 用于成本计算。

## 测试与验证
1.  **单元测试**: `backend/internal/collector/kubernetes_test.go` (验证 Slicing Mode 和 GPU 提取)。
2.  **集成测试**: `backend/scripts/test_api.sh` (验证 API 返回实时数据)。
3.  **模拟数据**: `backend/scripts/mock_data.go` (清空并重置演示环境)。
