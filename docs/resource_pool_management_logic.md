# AIPower-Efficiency-Pilot 资源池资产管理实现逻辑

## 1. 核心理念：从“标签”到“资产” (Label to Asset)
在 K8s 底层，资源池仅仅是 Node 上的一个 Label（如 `nvidia.com/pool-id`）。在 FinOps 体系中，我们需要将这个“字符串”转化为一个具有业务属性的“资产实体”，从而支持成本分摊、效能分析与场景对标。

## 2. 资产发现流转图 (Data Flow)

```mermaid
graph TD
    A[K8s Node Label] -->|Collector 感知| B{数据库是否存在 Pool-ID?}
    B -->|不存在| C[自动创建 Asset 存根]
    B -->|已存在| D[更新硬指标: 型号/容量]
    C --> E[管理员补充业务元数据]
    D --> E
    E --> F[FinOps 综合看板展示]
```

## 3. 实现细节

### 3.1 自动注册逻辑 (Auto-Registration)
当 `K8sCollector` 监听到 Node 的 Add/Update 事件时：
1.  提取标签 `nvidia.com/pool-id`。
2.  提取硬件信息 `nvidia.com/gpu.product` 和切分模式。
3.  调用 `UpsertResourcePool` 方法：
    *   **Insert**: 若为新池子，写入 `pool_id`、`gpu_model`，并将 `name` 默认设为 `pool_id`。
    *   **Update**: 若已存在，仅同步最新的硬件型号与切分状态，保留管理员手动修改的业务名称和描述。

### 3.2 业务元数据模型 (Metadata Schema)
| 字段 | 来源 | 说明 |
| :--- | :--- | :--- |
| **Pool ID** | K8s Label | 唯一标识（主键），如 `Infer-A100-MIG-Pool` |
| **业务别名** | 手动录入 | 友好名称，如“搜索中心核心推理池” |
| **业务场景** | 手动录入 | 场景：大模型预训练、模型微调、核心推理、小模型推理、研发调试 |
| **硬件型号** | 自动感知 | 真实型号，如 `NVIDIA A100-SXM4-80GB` |
| **硬件特性** | 自动感知 | 关键能力：NVLink, RDMA, TF32, FP8, Multi-Instance GPU 等 |
| **切分模式** | 自动感知 | 虚拟化技术：Full, MIG, MPS, TS |
| **定价逻辑** | 手动录入 | 财务属性：资源预留 (Reserved), 按规格计费, 吞吐量分摊, 极低单价 (Spot) |
| **治理优先级** | 手动录入 | 治理权重：生产级 (High), 测试级 (Low) |
| **资产描述** | 手动录入 | 补充说明，如“2024年三期采购，主要用于 LLM 提效” |

### 3.3 与计费引擎的关联
*   **定价索引**：计费引擎通过 `pool_id` 在 `pool_pricing` 表中查找单价。
*   **效能量化**：资源池资产管理提供的“业务场景”标签，可用于在看板上进行“同类场景利用率对标”。

## 4. 管理操作
*   **资产盘点**：提供全局视图，查看当前 K8s 集群中实际存在的物理池。
*   **信息补全**：管理员通过 UI 界面，将技术侧的 Pool-ID 与财务侧的成本中心完成映射。
