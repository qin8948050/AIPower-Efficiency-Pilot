# AIPower-Efficiency-Pilot 计费与聚合流程详解 (Phase 2)

> **业务目标：**
> 1. **💰 池子维度**：量化各资源池（A100/V100/H800）的真实 GPU 利用率与日均成本，支撑采购策略与算力定价决策。
> 2. **🏢 业务维度**：按团队（`team_label`）/ 项目（`project_label`）下钻，产出部门级成本分摊报告，解决"钱花到哪里去了"的问题。
> 3. **🔬 Pod 维度**：精确追溯每个训练/推理任务的利用率、显存峰值与实际费用，识别资源浪费的高危任务。

---

## 步骤一：会话级指标缝合 (Session Metrics Stitching)

*   **是什么：** 在 T+1 阶段，将 Phase 1 记录的 Pod 生命周期窗口，与 Prometheus 中的历史时序指标进行异步关联与核算。
*   **为什么：** 
    - Pod 销毁后 Prometheus 标签立刻失效，无法实时回查。
    - 离线缝合可以做到 100% 的指标覆盖率，且不影响生产链路。
*   **实现逻辑：**
    1.  **查找待缝合记录：** 定时扫描 MySQL `life_trace` 表，过滤出 `end_time IS NOT NULL AND gpu_util_avg = 0`（已结束但未缝合）的记录。
    2.  **业务归属提取（Phase 1 增补）：** Collector 在感知 Pod 启动时，同步提取 Pod Labels 中的 `app.kubernetes.io/team` 和 `app.kubernetes.io/project`，写入 `life_trace.team_label` / `project_label`，作为业务维度下钻的锚点。
    3.  **时序回溯查询：** 以 `life_trace.start_time` ~ `end_time` 为区间，向 Prometheus 发起 `range_query`，查询：
        - `DCGM_FI_DEV_GPU_UTIL`：算力利用率
        - `DCGM_FI_DEV_FB_USED`：显存用量 (MiB)
        - `DCGM_FI_DEV_POWER_USAGE`：功耗 (W)
    3.  **聚合计算：** 计算窗口内的 `Avg`（均值）、`Max`（峰值），得到：
        - `gpu_util_avg`, `gpu_util_max`
        - `mem_used_max`
        - `power_usage_avg`
    4.  **持久化回填：** 调用 `UpdateLifeTraceMetrics` 将结果写回 MySQL `life_trace` 表对应记录。
*   **效果：** 每一条 `life_trace` 记录从"只有时间戳"进化为"带有完整指标的计费凭据"。

---

## 步骤二：资源池定价引擎 (Pricing Engine)

*   **是什么：** 一套基于"资源池类型"和"虚拟化模式"双维度的弹性定价模型。
*   **为什么：** 
    - 不同型号 GPU（V100, A100, H800）采购成本差异悬殊，不能用统一单价计费。
    - 同一块 GPU 使用 MIG 切割后，不同切片大小的计费权重也应不同（非线性分摊）。
*   **数据模型 (`pool_pricing` 表)：**

    | 字段名 | 类型 | 说明 |
    |---|---|---|
    | `pool_id` | VARCHAR | 资源池 ID（主键关联 `life_trace`）|
    | `gpu_model` | VARCHAR | GPU 型号（V100/A100/H800）|
    | `base_price_per_hour` | DECIMAL | 基准小时单价（元/小时）|
    | `slicing_weights` | JSON | 各切片模式的权重系数 `{"Full":1.0, "MIG":0.35, "MPS":0.5, "TS":0.6}` |

*   **计费公式：**
    ```
    计费时长 (H)   = (end_time - start_time) / 3600
    切片权重        = pool_pricing.slicing_weights[life_trace.slicing_mode]
    成本 (元)       = 计费时长 × base_price_per_hour × 切片权重
    ```
*   **效果：** 为每一条 `life_trace` 写入 `cost_amount` 字段，产出精确到秒的成本凭据。

---

## 步骤三：1d 聚合引擎 (Daily Aggregation Engine)

*   **是什么：** 每日凌晨运行的批处理任务，将当天所有已完成的 `life_trace` 记录聚合为一张统一的"日级账单快照"。
*   **为什么：**
    - 前端 Dashboard 和账单报表需要按天、按周、按月聚合的数据，而非毫秒级的 Pod 记录。
    - 日级快照方便做同比环比趋势分析，且大幅降低实时查询的 DB 压力。
*   **数据模型 (`daily_billing_snapshot` 表)：**

    | 字段名 | 类型 | 说明 |
    |---|---|---|
    | `snapshot_date` | DATE | 快照日期（唯一索引）|
    | `pool_id` | VARCHAR | 资源池 ID |
    | `namespace` | VARCHAR | 业务命名空间 |
    | `total_cost` | DECIMAL | 当日消耗总金额 |
    | `avg_util_p95` | FLOAT | 当日 P95 利用率 |
    | `max_mem_gib` | FLOAT | 当日峰值显存 |
    | `pod_session_count` | INT | 当日任务会话数 |

*   **执行逻辑：**
    1.  按 `snapshot_date = 昨天` 过滤 `life_trace`。
    2.  按 `pool_id` + `namespace` 分组聚合（SUM cost, P95 util, MAX mem）。
    3.  写入或 Upsert 到 `daily_billing_snapshot`。
*   **效果：** 前端查询成本趋势时，直接读取此表，响应时间从秒级降至毫秒级。

---

## 步骤四：成本分摊看板 API (Billing Dashboard API)

*   **是什么：** 对外提供聚合账单数据的 REST API，供前端成本中心看板使用。
*   **接口规划：**

    | Method | Path | 说明 |
    |---|---|---|
    | GET | `/api/v2/billing/daily` | 查询每日账单汇总（按 Pool 或 Namespace 分组）|
    | GET | `/api/v2/billing/sessions` | 查询 Pod 会话级账单明细 |
    | GET | `/api/v2/pricing` | 查询各资源池当前定价配置 |
    | PUT | `/api/v2/pricing/:pool_id` | 管理员更新指定池子的单价及权重 |

---

## 验证计划

*   **单元测试：**
    - 计费公式测试：给定时长+池子+切片模式，验证金额计算是否符合预期。
    - 聚合逻辑测试：给定多条 `life_trace`，验证日级快照生成的正确性。
*   **集成测试（Mock Prometheus）：**
    - 启动 Mock Prometheus Server，注入假的 `range_query` 响应，验证指标缝合流程端到端正确性。
*   **端到端演示：**
    - 运行 `mock_step4_stitch.go` 验证 MySQL `life_trace` 记录是否被成功回填成本与指标。
    - 调用 `/api/v2/billing/daily` 验证 API 能否正常返回聚合账单。
