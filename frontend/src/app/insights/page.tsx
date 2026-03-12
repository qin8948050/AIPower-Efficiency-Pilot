"use client";

import { useEffect, useState } from "react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  BrainCircuit,
  Loader2,
  RefreshCw,
  CheckCircle,
  XCircle,
  AlertTriangle,
  TrendingDown,
  FileText,
  DollarSign,
} from "lucide-react";

interface InsightReport {
  id: string;
  generated_at: string;
  task_name: string;   // 任务名（Pod/PyTorchJob）
  namespace: string;   // 命名空间
  team: string;        // 负责团队
  pool_id: string;    // 当前所在资源池
  report_type: string;
  summary: string;
  root_cause: string;
  actions: string;
  est_savings: number;
  status: string;
}

interface ReportListResponse {
  reports: InsightReport[];
  total: number;
}

export default function InsightsPage() {
  const [reports, setReports] = useState<InsightReport[]>([]);
  const [loading, setLoading] = useState(false);
  const [generating, setGenerating] = useState(false);
  const [selectedReport, setSelectedReport] = useState<InsightReport | null>(null);

  useEffect(() => {
    fetchReports();
  }, []);

  const fetchReports = async () => {
    setLoading(true);
    try {
      const res = await fetch("http://localhost:8080/api/v3/insights/reports?limit=20");
      const data: ReportListResponse = await res.json();
      setReports(data.reports || []);
    } catch (error) {
      console.error("Failed to fetch reports:", error);
    } finally {
      setLoading(false);
    }
  };

  const generateReport = async () => {
    setGenerating(true);
    try {
      const res = await fetch("http://localhost:8080/api/v3/insights/generate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ days: 7 }),
      });
      const newReport: InsightReport = await res.json();
      setReports([newReport, ...reports]);
    } catch (error) {
      console.error("Failed to generate report:", error);
    } finally {
      setGenerating(false);
    }
  };

  const updateStatus = async (id: string, status: string) => {
    try {
      await fetch(`http://localhost:8080/api/v3/insights/reports/${id}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status }),
      });
      setReports(reports.map((r) => (r.id === id ? { ...r, status } : r)));
      if (selectedReport?.id === id) {
        setSelectedReport({ ...selectedReport, status });
      }
    } catch (error) {
      console.error("Failed to update status:", error);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case "approved":
        return (
          <Badge className="bg-green-500">
            <CheckCircle className="w-3 h-3 mr-1" />
            已批准
          </Badge>
        );
      case "rejected":
        return (
          <Badge className="bg-red-500">
            <XCircle className="w-3 h-3 mr-1" />
            已拒绝
          </Badge>
        );
      default:
        return (
          <Badge className="bg-yellow-500">
            <AlertTriangle className="w-3 h-3 mr-1" />
            待审批
          </Badge>
        );
    }
  };

  const getReportTypeBadge = (type: string) => {
    switch (type) {
      case "downgrade":
        return (
          <Badge variant="outline" className="text-orange-500 border-orange-500">
            降级迁移
          </Badge>
        );
      case "isolation":
        return (
          <Badge variant="outline" className="text-blue-500 border-blue-500">
            稳定性升级
          </Badge>
        );
      default:
        return <Badge variant="outline">一般建议</Badge>;
    }
  };

  const parseActions = (actionsStr: string) => {
    try {
      return JSON.parse(actionsStr);
    } catch {
      return [];
    }
  };

  // 从 Actions 中提取任务名（Pod 名称）
  const extractTaskName = (actionsStr: string): string => {
    const actions = parseActions(actionsStr);
    if (actions.length > 0 && actions[0].pod_name) {
      // 提取任务前缀（如 pytorchjob-cv-train）而不是完整的 worker 名称
      const podName = actions[0].pod_name;
      const prefix = podName.split('-worker-')[0];
      return prefix;
    }
    return "";
  };

  // 从 Actions 中提取命名空间
  const extractNamespace = (actionsStr: string): string => {
    const actions = parseActions(actionsStr);
    if (actions.length > 0 && actions[0].namespace) {
      return actions[0].namespace;
    }
    return "";
  };

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const pendingCount = reports.filter((r) => r.status === "pending").length;
  const approvedCount = reports.filter((r) => r.status === "approved").length;
  const totalSavings = reports.reduce((sum, r) => sum + r.est_savings, 0);

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      {/* Header */}
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <BrainCircuit className="h-8 w-8 text-purple-600" />
          AI 诊断报告
        </h2>
        <div className="flex items-center space-x-2">
          <Button variant="outline" size="sm" onClick={fetchReports} disabled={loading}>
            {loading ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <RefreshCw className="h-4 w-4 mr-2" />}
            刷新
          </Button>
          <Button size="sm" onClick={generateReport} disabled={generating}>
            {generating ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <TrendingDown className="h-4 w-4 mr-2" />}
            生成诊断报告
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总报告数</CardTitle>
            <FileText className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{reports.length}</div>
            <p className="text-xs text-muted-foreground mt-1">历史诊断记录</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">待审批</CardTitle>
            <AlertTriangle className="h-4 w-4 text-yellow-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">{pendingCount}</div>
            <p className="text-xs text-muted-foreground mt-1">待人工确认</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">已批准</CardTitle>
            <CheckCircle className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{approvedCount}</div>
            <p className="text-xs text-muted-foreground mt-1">已执行治理</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">预计年度节省</CardTitle>
            <DollarSign className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">${totalSavings.toFixed(2)}</div>
            <p className="text-xs text-muted-foreground mt-1">预期成本节省</p>
          </CardContent>
        </Card>
      </div>

      {/* Reports Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        {/* Report List */}
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>诊断报告列表</CardTitle>
            <CardDescription className="text-sm text-muted-foreground">
              基于 LLM 的智能效能分析与优化建议
            </CardDescription>
          </CardHeader>
          <CardContent>
            {reports.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                暂无诊断报告，请点击"生成诊断报告"创建
              </div>
            ) : (
              <div className="space-y-3">
                {reports.map((report) => (
                  <div
                    key={report.id}
                    className={`p-4 border rounded-lg cursor-pointer transition-all hover:bg-slate-50 ${
                      selectedReport?.id === report.id ? "border-primary bg-slate-50" : ""
                    }`}
                    onClick={() => setSelectedReport(report)}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        {getReportTypeBadge(report.report_type)}
                        {getStatusBadge(report.status)}
                      </div>
                      <span className="text-xs text-muted-foreground">
                        {formatDate(report.generated_at)}
                      </span>
                    </div>
                    <div className="font-medium">
                      {report.task_name || report.pool_id}
                      <span className="text-muted-foreground font-normal text-sm ml-2">
                        {report.namespace && `(${report.namespace})`}
                      </span>
                    </div>
                    <div className="text-sm text-muted-foreground line-clamp-1 mt-1">
                      {report.summary}
                    </div>
                    <div className="text-sm mt-2">
                      {report.est_savings > 0 ? (
                        <>
                          <span className="text-muted-foreground">预计节省: </span>
                          <span className="font-semibold text-green-600">
                            ${report.est_savings.toFixed(2)}/年
                          </span>
                        </>
                      ) : (
                        <>
                          <span className="text-muted-foreground">预计增加: </span>
                          <span className="font-semibold text-red-600">
                            ${Math.abs(report.est_savings).toFixed(2)}/年
                          </span>
                        </>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </CardContent>
        </Card>

        {/* Detail Panel */}
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>报告详情</CardTitle>
            {selectedReport && (
              <CardDescription className="text-sm">
                任务: {selectedReport.task_name || "未知"} ({selectedReport.namespace}) | 团队: {selectedReport.team || "未知"} | 资源池: {selectedReport.pool_id}
              </CardDescription>
            )}
          </CardHeader>
          <CardContent>
            {selectedReport ? (
              <div className="space-y-6">
                {/* Status Badge */}
                <div className="flex items-center justify-between">
                  {getStatusBadge(selectedReport.status)}
                </div>

                {/* Summary */}
                <div>
                  <h4 className="text-sm font-semibold mb-2">摘要</h4>
                  <p className="text-sm text-muted-foreground">{selectedReport.summary}</p>
                </div>

                {/* Root Cause */}
                <div>
                  <h4 className="text-sm font-semibold mb-2">根因分析</h4>
                  <p className="text-sm text-muted-foreground">{selectedReport.root_cause}</p>
                </div>

                {/* Actions */}
                <div>
                  <h4 className="text-sm font-semibold mb-2">优化动作</h4>
                  <div className="space-y-2">
                    {parseActions(selectedReport.actions).map((action: any, idx: number) => (
                      <div key={idx} className="p-3 bg-muted rounded-lg text-sm">
                        <div className="flex items-center justify-between mb-1">
                          <Badge variant="outline">{action.type}</Badge>
                          <span className="font-mono text-xs">
                            {action.from_pool} → {action.to_pool}
                          </span>
                        </div>
                        <div className="text-muted-foreground">
                          {action.namespace}/{action.pod_name}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* Est Savings / Cost Increase */}
                {selectedReport.est_savings > 0 ? (
                  <div className="p-4 bg-green-50 rounded-lg">
                    <div className="flex items-center gap-2 text-green-700">
                      <TrendingDown className="h-5 w-5" />
                      <span className="font-semibold">预期年度节省</span>
                    </div>
                    <div className="text-2xl font-bold text-green-700 mt-1">
                      ${selectedReport.est_savings.toFixed(2)}
                    </div>
                  </div>
                ) : (
                  <div className="p-4 bg-red-50 rounded-lg">
                    <div className="flex items-center gap-2 text-red-700">
                      <TrendingDown className="h-5 w-5 rotate-180" />
                      <span className="font-semibold">预期年度增加</span>
                    </div>
                    <div className="text-2xl font-bold text-red-700 mt-1">
                      ${Math.abs(selectedReport.est_savings).toFixed(2)}
                    </div>
                  </div>
                )}

                {/* Actions Buttons */}
                {selectedReport.status === "pending" && (
                  <div className="flex gap-2 pt-4 border-t">
                    <Button
                      className="flex-1 bg-green-600 hover:bg-green-700"
                      onClick={() => updateStatus(selectedReport.id, "approved")}
                    >
                      <CheckCircle className="h-4 w-4 mr-2" />
                      批准执行
                    </Button>
                    <Button
                      variant="destructive"
                      className="flex-1"
                      onClick={() => updateStatus(selectedReport.id, "rejected")}
                    >
                      <XCircle className="h-4 w-4 mr-2" />
                      拒绝
                    </Button>
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                选择左侧报告查看详情
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
