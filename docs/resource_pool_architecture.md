# AIPower-Efficiency-Pilot 资源池矩阵架构说明书

## 1. 架构设计核心：多维属性命名体系
为了实现财务透明、运维高效与调度精准，本项目采用 `[场景]-[型号]-[切分标识]-Pool` 的四段式命名规范。

## 2. 资源池命名矩阵 (The Naming Matrix)

| 资源池 ID (Pool ID) | 硬件特性 (Hardware) | **虚拟化切分 (Slicing Mode)** | 业务场景 | 定价逻辑 |
| :--- | :--- | :--- | :--- | :--- |
| **Train-H800-Full-Pool** | H800 NVLink | **Full (独享整卡)** | 大模型预训练 | 资源预留 (Reserved) |
| **Train-A100-Full-Pool** | A100 NVLink | **Full (独享整卡)** | 模型全量微调 | 资源预留 (Reserved) |
| **Infer-A100-MIG-Pool** | A100 | **MIG (硬件级切分)** | 核心大模型推理 | 按切片规格计费 |
| **Infer-L4-MPS-Pool** | L4 | **MPS (并行切片)** | 小模型/高并发推理 | 吞吐量分摊计费 |
| **Dev-T4-TS-Pool** | T4 / 旧卡 | **TS (Time-Slicing/超分)** | 研发调试/单元测试 | 极低单价 (Spot) |

## 3. 业务流转与治理逻辑

### 3.1 成本引擎 (Cost Engine)
*   **计费逻辑自动匹配：** 计费引擎根据 `Slicing Mode` 自动应用不同的成本分摊算法。
    *   `Full`：分摊整张卡成本。
    *   `MIG`：分摊特定 Profile 的硬件分片成本。
    *   `TS`：应用超分折扣。

### 3.2 智能调优 (Right-sizing)
*   **虚拟化模式迁移建议：**
    *   *案例：* 发现 `MPS-Pool` 中的任务经常发生显存溢出风险，AI 建议：*“该任务显存占用具有突发性，建议迁移至 **MIG-Pool** 以获得更强的硬件级隔离保护。”*
    *   *案例：* 发现 `Full-Pool` 里的 A100 只用了 5G 显存，AI 建议：*“建议降级至 **L4-MPS-Pool** 以节省 80% 成本。”*

## 4. 实施机制
*   **Taints & Labels:** `nvidia.com/slicing-mode: mig`, `nvidia.com/pool-name: Infer-A100-MIG-Pool`。
*   **Priority:** 生产级池 (Full/MIG) 优先级远高于研发池 (TS)。
