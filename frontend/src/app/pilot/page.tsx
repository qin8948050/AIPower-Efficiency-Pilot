"use client";

import { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  TrendingDown,
  Loader2,
  XCircle,
  CheckCircle,
  Clock,
  AlertCircle,
  RefreshCw,
  ShieldCheck,
} from "lucide-react";

interface GovernanceExecution {
  id: number;
  report_id: number;
  task_name: string;
  namespace: string;
  action_type: string;
  from_pool: string;
  to_pool: string;
  from_gpu: number;
  to_gpu: number;
  patch_type?: string;
  patch_content?: string;
  status: string;
  scheduled_at?: string;
  executed_at?: string;
  error_msg?: string;
  created_at: string;
}

interface GovernanceStats {
  total_executed: number;
  pending_count: number;
}

interface ExecutionsResponse {
  executions: GovernanceExecution[];
  total: number;
}

export default function PilotPage() {
  const [executions, setExecutions] = useState<GovernanceExecution[]>([]);
  const [stats, setStats] = useState<GovernanceStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [filterStatus, setFilterStatus] = useState<string>("");
  const [selectedExec, setSelectedExec] = useState<GovernanceExecution | null>(null);

  useEffect(() => {
    fetchExecutions();
    fetchStats();
  }, [filterStatus]);

  const fetchExecutions = async () => {
    setLoading(true);
    try {
      const statusParam = filterStatus ? `&status=${filterStatus}` : "";
      const res = await fetch(`http://localhost:8080/api/v4/governance/executions?limit=50${statusParam}`);
      if (!res.ok) throw new Error(`HTTP error: ${res.status}`);
      const data: ExecutionsResponse = await res.json();
      setExecutions(data.executions || []);
    } catch (error) {
      console.error("Failed to fetch:", error);
      setExecutions([]);
    } finally {
      setLoading(false);
    }
  };

  const fetchStats = async () => {
    try {
      const res = await fetch("http://localhost:8080/api/v4/governance/stats");
      if (!res.ok) throw new Error(`HTTP error: ${res.status}`);
      const data: GovernanceStats = await res.json();
      setStats(data);
    } catch (error) {
      console.error("Failed to fetch stats:", error);
    }
  };

  const cancelExecution = async (id: number) => {
    try {
      await fetch(`http://localhost:8080/api/v4/governance/executions/${id}/cancel`, { method: "PUT" });
      fetchExecutions();
      fetchStats();
    } catch (error) {
      console.error("Failed to cancel:", error);
    }
  };

  const getStatusBadge = (status: string): React.ReactNode => {
    const map: Record<string, { label: string; color: string; icon: React.ReactNode }> = {
      pending: { label: "待执行", color: "bg-yellow-500", icon: <Clock className="w-3 h-3 mr-1" /> },
      executing: { label: "执行中", color: "bg-blue-500", icon: <Loader2 className="w-3 h-3 mr-1 animate-spin" /> },
      completed: { label: "已完成", color: "bg-green-500", icon: <CheckCircle className="w-3 h-3 mr-1" /> },
      failed: { label: "失败", color: "bg-red-500", icon: <AlertCircle className="w-3 h-3 mr-1" /> },
      cancelled: { label: "已取消", color: "bg-gray-500", icon: <XCircle className="w-3 h-3 mr-1" /> },
    };
    const s = map[status] || { label: status, color: "bg-gray-500", icon: null };
    return <Badge className={s.color}>{s.icon}{s.label}</Badge>;
  };

  const getActionLabel = (t: string) => ({ downgrade: "降配", migrate: "迁移", downgrade_migrate: "降配+迁移" }[t] || t);
  const formatDate = (s: string) => s ? new Date(s).toLocaleString("zh-CN") : "-";
  const formatJson = (s: string) => { try { return JSON.stringify(JSON.parse(s), null, 2); } catch { return s; } };

  const completedCount = executions.filter(e => e.status === "completed").length;
  const failedCount = executions.filter(e => e.status === "failed").length;

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold flex items-center gap-2">
          <TrendingDown className="h-8 w-8 text-indigo-600" />治理执行中心
        </h2>
        <Button variant="outline" size="sm" onClick={fetchExecutions} disabled={loading}>
          {loading ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <RefreshCw className="h-4 w-4 mr-2" />}刷新
        </Button>
      </div>

      <div className="grid gap-4 md:grid-cols-4">
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">已执行治理</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold">{stats?.total_executed ?? completedCount}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">待执行</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold text-yellow-600">{stats?.pending_count ?? 0}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">执行成功</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold text-green-600">{completedCount}</div></CardContent></Card>
        <Card><CardHeader className="pb-2"><CardTitle className="text-sm">执行失败</CardTitle></CardHeader><CardContent><div className="text-2xl font-bold text-red-600">{failedCount}</div></CardContent></Card>
      </div>

      <select className="border rounded px-3 py-1.5 text-sm" value={filterStatus} onChange={e => setFilterStatus(e.target.value)}>
        <option value="">全部状态</option>
        <option value="pending">待执行</option>
        <option value="executing">执行中</option>
        <option value="completed">已完成</option>
        <option value="failed">失败</option>
        <option value="cancelled">已取消</option>
      </select>

      {loading ? <div className="flex justify-center py-12"><Loader2 className="h-8 w-8 animate-spin" /></div> : executions.length === 0 ? (
        <Card><CardContent className="py-12 text-center text-muted-foreground"><TrendingDown className="h-12 w-12 mx-auto mb-4" /><p>暂无治理执行记录</p></CardContent></Card>
      ) : (
        <div className="space-y-2">
          {executions.map(exec => (
            <Card key={exec.id} className={`cursor-pointer hover:shadow-md ${selectedExec?.id === exec.id ? "ring-2 ring-primary" : ""}`} onClick={() => setSelectedExec(exec)}>
              <CardContent className="p-4 flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <span className="font-medium">{exec.task_name}</span>
                  {getStatusBadge(exec.status)}
                  <span className="text-sm text-muted-foreground">{exec.namespace}</span>
                </div>
                <div className="flex items-center gap-4 text-sm">
                  <span>{getActionLabel(exec.action_type)}</span>
                  <span className="text-muted-foreground">{exec.from_gpu}→{exec.to_gpu} GPU</span>
                  <span className="text-muted-foreground hidden md:inline">{exec.from_pool}→{exec.to_pool}</span>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* 详情弹窗 */}
      {selectedExec && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 bg-black/50" onClick={() => setSelectedExec(null)} />
          <div className="relative bg-background rounded-lg shadow-xl w-[700px] max-h-[85vh] overflow-hidden">
            <div className="p-4 border-b flex items-center justify-between">
              <div className="flex items-center gap-3">
                <h3 className="text-lg font-semibold">执行详情</h3>
                {getStatusBadge(selectedExec.status)}
              </div>
              <Button variant="ghost" size="sm" onClick={() => setSelectedExec(null)}>✕</Button>
            </div>
            <div className="p-5 overflow-y-auto max-h-[calc(85vh-60px)] space-y-4">
              <div className="grid grid-cols-2 gap-4 text-sm">
                <div><span className="text-muted-foreground">任务名称</span><div className="font-medium">{selectedExec.task_name}</div></div>
                <div><span className="text-muted-foreground">命名空间</span><div>{selectedExec.namespace}</div></div>
                <div><span className="text-muted-foreground">动作类型</span><div>{getActionLabel(selectedExec.action_type)}</div></div>
                <div><span className="text-muted-foreground">GPU变更</span><div>{selectedExec.from_gpu} → {selectedExec.to_gpu}</div></div>
              </div>
              <div>
                <span className="text-muted-foreground text-sm">资源池变更</span>
                <div className="p-3 bg-muted rounded mt-1 text-sm">{selectedExec.from_pool} → {selectedExec.to_pool}</div>
              </div>
              {selectedExec.patch_content && (
                <div>
                  <div className="font-medium mb-2 text-sm">Patch 内容：</div>
                  <pre className="p-3 bg-slate-900 text-slate-50 rounded text-sm overflow-auto max-h-[280px] whitespace-pre-wrap break-all">{formatJson(selectedExec.patch_content)}</pre>
                </div>
              )}
              {selectedExec.status === "failed" && selectedExec.error_msg && <div className="p-3 bg-red-50 text-red-700 rounded">{selectedExec.error_msg}</div>}
              {(selectedExec.status === "pending" || selectedExec.status === "executing") && (
                <Button variant="destructive" className="w-full" onClick={() => { cancelExecution(selectedExec.id); setSelectedExec(null); }}>取消执行</Button>
              )}
            </div>
          </div>
        </div>
      )}

      <Card>
        <CardHeader><CardTitle>使用说明</CardTitle></CardHeader>
        <CardContent className="text-sm text-muted-foreground space-y-1">
          <p>1. 在"AI诊断报告"批准建议后执行治理</p>
          <p>2. 降配直接修改Pod GPU限制，迁移添加标签由调度器处理</p>
          <p className="text-yellow-600">生产环境请谨慎操作</p>
        </CardContent>
      </Card>
    </div>
  );
}
