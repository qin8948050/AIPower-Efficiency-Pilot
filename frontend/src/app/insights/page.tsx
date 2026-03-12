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
  Play,
} from "lucide-react";

interface Recommendation {
  action_type: string;  // 动作类型: 降配, 迁移, 降配+迁移
  from_gpu: number;
  to_gpu: number;
  from_pool: string;
  to_pool: string;
  est_savings: number;
  reason: string;
}

interface InsightReport {
  id: string;
  generated_at: string;
  task_name: string;   // 任务名
  namespace: string;   // 命名空间
  team: string;        // 负责团队
  pool_id: string;    // 当前所在资源池
  problem: string;    // 问题描述: 利用率低, 抖动高
  report_type: string;
  summary: string;
  root_cause: string;
  recommendations: string;  // JSON 字符串，建议列表
  approved_recommendation?: string;  // 批准的建议
  approved_at?: string;  // 批准时间
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
  const [selectedRecIndex, setSelectedRecIndex] = useState<number | null>(null);

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
      // 如果是批准，需要包含选中的建议
      const body: any = { status };
      if (status === "approved" && selectedRecIndex !== null && selectedReport) {
        const recs = parseRecommendations(selectedReport.recommendations);
        if (recs[selectedRecIndex]) {
          body.recommendation = JSON.stringify(recs[selectedRecIndex]);
        }
      }
      await fetch(`http://localhost:8080/api/v3/insights/reports/${id}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      setReports(reports.map((r) => (r.id === id ? { ...r, status } : r)));
      if (selectedReport?.id === id) {
        setSelectedReport({ ...selectedReport, status });
      }
      setSelectedRecIndex(null);
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

  const parseRecommendations = (recommendationsStr: string) => {
    try {
      return JSON.parse(recommendationsStr);
    } catch {
      return [];
    }
  };

  // 获取问题类型的显示文本
  const getProblemText = (problem: string): string => {
    switch (problem) {
      case "利用率低":
        return "利用率低";
      case "抖动高":
        return "抖动高";
      case "特性不匹配":
        return "特性不匹配";
      default:
        return problem || "一般问题";
    }
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
            <div className="text-2xl font-bold text-green-600">¥{totalSavings.toFixed(2)}</div>
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
                    onClick={() => {
                      setSelectedReport(report);
                      setSelectedRecIndex(null);
                    }}
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
                            ¥{report.est_savings.toFixed(2)}/年
                          </span>
                        </>
                      ) : (
                        <>
                          <span className="text-muted-foreground">预计增加: </span>
                          <span className="font-semibold text-red-600">
                            ¥{Math.abs(report.est_savings).toFixed(2)}/年
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

                {/* Problem */}
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">问题类型:</span>
                  <Badge variant={selectedReport.problem === "利用率低" ? "destructive" : "default"}>
                    {getProblemText(selectedReport.problem)}
                  </Badge>
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

                {/* Recommendations */}
                <div>
                  <h4 className="text-sm font-semibold mb-2">优化建议（单选）</h4>
                  <div className="space-y-2">
                    {(() => {
                      const recs = parseRecommendations(selectedReport.recommendations);
                      // 找出最佳建议：节省最多（或增加最少）
                      const bestRec = recs.length > 0 ? recs.reduce((best: Recommendation, curr: Recommendation) =>
                        curr.est_savings > best.est_savings ? curr : best
                      ) : null;

                      const isPending = selectedReport.status === "pending";
                      return recs.map((rec: Recommendation, idx: number) => {
                        const isBest = rec === bestRec && recs.length > 1;
                        const isSelected = selectedRecIndex === idx;
                        return (
                          <div
                            key={idx}
                            onClick={isPending ? () => setSelectedRecIndex(idx) : undefined}
                            className={`p-3 rounded-lg text-sm transition-all ${
                              isSelected
                                ? "bg-blue-50 border-2 border-blue-500"
                                : isBest
                                ? "bg-green-50 border-2 border-green-300"
                                : "bg-muted border border-border"
                            } ${isPending ? "cursor-pointer hover:border-gray-400" : "cursor-not-allowed opacity-75"}`}
                          >
                            <div className="flex items-center justify-between mb-1">
                              <div className="flex items-center gap-2">
                                {/* Radio indicator */}
                                <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                                  isSelected ? "border-blue-500 bg-blue-500" : "border-gray-400"
                                }`}>
                                  {isSelected && <div className="w-2 h-2 rounded-full bg-white" />}
                                </div>
                                <Badge variant="outline" className={
                                  rec.action_type === "降配" ? "border-orange-500 text-orange-500" :
                                  rec.action_type === "迁移" ? "border-blue-500 text-blue-500" :
                                  "border-purple-500 text-purple-500"
                                }>
                                  {rec.action_type}
                                </Badge>
                                {isBest && !isSelected && (
                                  <Badge className="bg-green-500 text-white text-xs">
                                    推荐
                                  </Badge>
                                )}
                              </div>
                              <span className="font-mono text-xs">
                                {rec.from_pool} → {rec.to_pool}
                              </span>
                            </div>
                            <div className="text-muted-foreground mb-1">
                              GPU: {rec.from_gpu} → {rec.to_gpu}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {rec.reason}
                            </div>
                            <div className="mt-2 pt-2 border-t border-border text-sm">
                              {rec.est_savings > 0 ? (
                                <span className="text-green-600 font-medium">
                                  节省: ¥{rec.est_savings.toFixed(2)}/年
                                </span>
                              ) : (
                                <span className="text-red-600 font-medium">
                                  增加: ¥{Math.abs(rec.est_savings).toFixed(2)}/年
                                </span>
                              )}
                            </div>
                          </div>
                        );
                      });
                    })()}
                  </div>
                </div>

                {/* Approved Recommendation Display */}
                {selectedReport.status === "approved" && selectedReport.approved_recommendation && (
                  <div className="pt-4 border-t">
                    <h4 className="text-sm font-semibold mb-2">已批准的建议</h4>
                    {(() => {
                      const approvedRec = parseRecommendations(`[${selectedReport.approved_recommendation}]`)[0];
                      if (!approvedRec) return null;

                      const actionTypeMap: Record<string, string> = {
                        "降配": "downgrade",
                        "迁移": "migrate",
                        "降配+迁移": "downgrade_migrate",
                      };

                      const handleExecute = async () => {
                        try {
                          const res = await fetch("http://localhost:8080/api/v4/governance/execute", {
                            method: "POST",
                            headers: { "Content-Type": "application/json" },
                            body: JSON.stringify({
                              report_id: parseInt(selectedReport.id),
                              task_name: selectedReport.task_name,
                              namespace: selectedReport.namespace,
                              action_type: actionTypeMap[approvedRec.action_type] || approvedRec.action_type,
                              from_pool: approvedRec.from_pool,
                              to_pool: approvedRec.to_pool,
                              from_gpu: approvedRec.from_gpu,
                              to_gpu: approvedRec.to_gpu,
                              execute_now: true,
                            }),
                          });
                          if (res.ok) {
                            alert("治理任务已提交执行，请前往「治理执行中心」查看进度");
                          }
                        } catch (error) {
                          console.error("Failed to execute:", error);
                          alert("执行失败，请重试");
                        }
                      };

                      return (
                        <div className="space-y-3">
                          <div className="p-3 bg-green-50 border-2 border-green-500 rounded-lg text-sm">
                            <div className="flex items-center justify-between mb-1">
                              <div className="flex items-center gap-2">
                                <CheckCircle className="h-4 w-4 text-green-600" />
                                <Badge variant="outline" className={
                                  approvedRec.action_type === "降配" ? "border-orange-500 text-orange-500" :
                                  approvedRec.action_type === "迁移" ? "border-blue-500 text-blue-500" :
                                  "border-purple-500 text-purple-500"
                                }>
                                  {approvedRec.action_type}
                                </Badge>
                              </div>
                              <span className="font-mono text-xs">
                                {approvedRec.from_pool} → {approvedRec.to_pool}
                              </span>
                            </div>
                            <div className="text-muted-foreground mb-1">
                              GPU: {approvedRec.from_gpu} → {approvedRec.to_gpu}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {approvedRec.reason}
                            </div>
                          </div>
                          <Button
                            className="w-full bg-blue-600 hover:bg-blue-700"
                            onClick={handleExecute}
                          >
                            <Play className="h-4 w-4 mr-2" />
                            立即执行治理
                          </Button>
                        </div>
                      );
                    })()}
                  </div>
                )}

                {/* Rejected Display */}
                {selectedReport.status === "rejected" && (
                  <div className="pt-4 border-t">
                    <div className="p-3 bg-red-50 border-2 border-red-500 rounded-lg text-sm flex items-center gap-2">
                      <XCircle className="h-4 w-4 text-red-600" />
                      <span className="text-red-700">已拒绝</span>
                    </div>
                  </div>
                )}

                {/* Actions Buttons */}
                {selectedReport.status === "pending" && (
                  <div className="flex flex-col gap-2 pt-4 border-t">
                    {parseRecommendations(selectedReport.recommendations).length > 0 && (
                      <div className="text-xs text-muted-foreground text-center">
                        请在上方选择一条建议后批准执行
                      </div>
                    )}
                    <div className="flex gap-2">
                      <Button
                        className="flex-1 bg-green-600 hover:bg-green-700"
                        disabled={selectedRecIndex === null}
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
                        拒绝全部
                      </Button>
                    </div>
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
