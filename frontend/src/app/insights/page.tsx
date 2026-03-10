"use client";

import { useEffect, useState } from "react";
import DashboardLayout from "../dashboard-layout";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Loader2, RefreshCw, CheckCircle, XCircle, AlertTriangle, TrendingDown } from "lucide-react";

interface InsightReport {
  id: string;
  generated_at: string;
  pool_id: string;
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
      const res = await fetch("/api/v3/insights/reports?limit=20");
      const data: ReportListResponse = await res.json();
      setReports(data.reports);
    } catch (error) {
      console.error("Failed to fetch reports:", error);
    } finally {
      setLoading(false);
    }
  };

  const generateReport = async () => {
    setGenerating(true);
    try {
      const res = await fetch("/api/v3/insights/generate", {
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
      await fetch(`/api/v3/insights/reports/${id}/status`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ status }),
      });
      setReports(reports.map(r => r.id === id ? { ...r, status } : r));
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
        return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" />已批准</Badge>;
      case "rejected":
        return <Badge className="bg-red-500"><XCircle className="w-3 h-3 mr-1" />已拒绝</Badge>;
      default:
        return <Badge className="bg-yellow-500"><AlertTriangle className="w-3 h-3 mr-1" />待审批</Badge>;
    }
  };

  const getReportTypeBadge = (type: string) => {
    switch (type) {
      case "downgrade":
        return <Badge variant="outline" className="text-orange-500 border-orange-500">降级迁移</Badge>;
      case "isolation":
        return <Badge variant="outline" className="text-blue-500 border-blue-500">稳定性升级</Badge>;
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

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <DashboardLayout>
    <div className="space-y-6">
      {/* 头部操作栏 */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">AI 诊断报告</h2>
          <p className="text-muted-foreground">基于 LLM 的智能效能分析与优化建议</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={fetchReports} disabled={loading}>
            {loading ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <RefreshCw className="w-4 h-4 mr-2" />}
            刷新
          </Button>
          <Button onClick={generateReport} disabled={generating}>
            {generating ? <Loader2 className="w-4 h-4 mr-2 animate-spin" /> : <TrendingDown className="w-4 h-4 mr-2" />}
            生成诊断报告
          </Button>
        </div>
      </div>

      {/* 统计卡片 */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总报告数</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{reports.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">待审批</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-yellow-600">
              {reports.filter(r => r.status === "pending").length}
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">预计年度节省</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">
              ${reports.reduce((sum, r) => sum + r.est_savings, 0).toFixed(2)}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 报告列表 */}
      <div className="grid gap-6 md:grid-cols-2">
        {/* 报告卡片列表 */}
        <div className="space-y-4">
          <h3 className="text-lg font-semibold">诊断报告列表</h3>
          {reports.length === 0 ? (
            <Card>
              <CardContent className="pt-6 text-center text-muted-foreground">
                暂无诊断报告，请点击"生成诊断报告"创建
              </CardContent>
            </Card>
          ) : (
            reports.map((report) => (
              <Card
                key={report.id}
                className={`cursor-pointer transition-all hover:shadow-md ${
                  selectedReport?.id === report.id ? "ring-2 ring-primary" : ""
                }`}
                onClick={() => setSelectedReport(report)}
              >
                <CardHeader className="pb-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      {getReportTypeBadge(report.report_type)}
                      {getStatusBadge(report.status)}
                    </div>
                    <span className="text-xs text-muted-foreground">
                      {formatDate(report.generated_at)}
                    </span>
                  </div>
                  <CardTitle className="text-base mt-2">{report.pool_id}</CardTitle>
                  <CardDescription className="line-clamp-2">{report.summary}</CardDescription>
                </CardHeader>
                <CardContent className="pt-0">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">预计节省</span>
                    <span className="font-semibold text-green-600">
                      ${report.est_savings.toFixed(2)}/年
                    </span>
                  </div>
                </CardContent>
              </Card>
            ))
          )}
        </div>

        {/* 报告详情面板 */}
        <div>
          {selectedReport ? (
            <Card className="sticky top-4">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle>报告详情</CardTitle>
                  {getStatusBadge(selectedReport.status)}
                </div>
                <CardDescription>
                  资源池: {selectedReport.pool_id} | 类型: {selectedReport.report_type}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-6">
                {/* 根因分析 */}
                <div>
                  <h4 className="font-semibold mb-2">根因分析</h4>
                  <p className="text-sm text-muted-foreground">{selectedReport.root_cause}</p>
                </div>

                {/* 优化动作 */}
                <div>
                  <h4 className="font-semibold mb-2">优化动作</h4>
                  <div className="space-y-2">
                    {parseActions(selectedReport.actions).map((action: any, idx: number) => (
                      <div key={idx} className="p-3 bg-muted rounded-lg text-sm">
                        <div className="flex items-center justify-between mb-1">
                          <Badge variant="outline">{action.type}</Badge>
                          <span className="font-mono text-xs">{action.from_pool} → {action.to_pool}</span>
                        </div>
                        <div className="text-muted-foreground">
                          {action.namespace}/{action.pod_name}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                {/* 预期收益 */}
                <div className="p-4 bg-green-50 rounded-lg">
                  <div className="flex items-center gap-2 text-green-700">
                    <TrendingDown className="w-5 h-5" />
                    <span className="font-semibold">预期年度节省</span>
                  </div>
                  <div className="text-2xl font-bold text-green-700 mt-1">
                    ${selectedReport.est_savings.toFixed(2)}
                  </div>
                </div>

                {/* 审批操作 */}
                {selectedReport.status === "pending" && (
                  <div className="flex gap-2 pt-4 border-t">
                    <Button
                      className="flex-1 bg-green-600 hover:bg-green-700"
                      onClick={() => updateStatus(selectedReport.id, "approved")}
                    >
                      <CheckCircle className="w-4 h-4 mr-2" />
                      批准执行
                    </Button>
                    <Button
                      variant="destructive"
                      className="flex-1"
                      onClick={() => updateStatus(selectedReport.id, "rejected")}
                    >
                      <XCircle className="w-4 h-4 mr-2" />
                      拒绝
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
          ) : (
            <Card>
              <CardContent className="pt-6 text-center text-muted-foreground h-64 flex items-center justify-center">
                选择左侧报告查看详情
              </CardContent>
            </Card>
          )}
        </div>
      </div>
    </div>
    </DashboardLayout>
  );
}
