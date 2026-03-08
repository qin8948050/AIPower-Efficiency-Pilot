"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  BrainCircuit,
  Cpu,
  Database,
  TrendingDown,
  Activity,
  RefreshCcw,
} from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";

interface PoolStat {
  id: string;
  slicing_mode: string;
  pod_count: number;
  gpu_util_avg: number;
  mem_used_bytes: number;
}

interface PodTrace {
  namespace: string;
  pod_name: string;
  pool_id: string;
  slicing_mode: string;
  metrics?: {
    gpu_util_avg: number;
    mem_used_bytes: number;
    mem_total_bytes: number;
  };
}

export default function DashboardPage() {
  const [pools, setPools] = useState<PoolStat[]>([]);
  const [traces, setTraces] = useState<PodTrace[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastRefreshed, setLastRefreshed] = useState<Date>(new Date());

  const fetchData = async () => {
    try {
      setLoading(true);
      const [poolsRes, tracesRes] = await Promise.all([
        fetch("http://localhost:8080/api/v1/pools"),
        fetch("http://localhost:8080/api/v1/traces"),
      ]);

      const poolsData = await poolsRes.json();
      const tracesData = await tracesRes.json();

      setPools(poolsData || []);
      setTraces(tracesData || []);
      setLastRefreshed(new Date());
    } catch (error) {
      console.error("Failed to fetch dashboard data:", error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchData();
    const timer = setInterval(fetchData, 10000); // 10秒自动刷新
    return () => clearInterval(timer);
  }, []);

  // 计算看板指标
  const totalPods = traces.length;
  const avgUtil = pools.length > 0
    ? (pools.reduce((acc, p) => acc + p.gpu_util_avg, 0) / pools.length).toFixed(1)
    : "0.0";
  const activePools = pools.length;

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 pt-0">
      <div className="flex items-center justify-between">
        <h2 className="text-lg font-semibold flex items-center gap-2">
          <Activity className="h-5 w-5 text-primary" />
          系统活动概览
        </h2>
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span>上次刷新: {lastRefreshed.toLocaleTimeString()}</span>
          <Button variant="ghost" size="icon" className="h-8 w-8" onClick={fetchData}>
            <RefreshCcw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </Button>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="bg-gradient-to-br from-background to-muted/50 border-primary/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">活跃资源池</CardTitle>
            <Database className="h-4 w-4 text-primary" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{activePools}</div>
            <p className="text-xs text-muted-foreground mt-1">
              当前集群下运行中的 Pool
            </p>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-background to-muted/50 border-primary/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">当前平均利用率</CardTitle>
            <Cpu className="h-4 w-4 text-primary" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{avgUtil}%</div>
            <div className="flex items-center text-xs text-green-500 mt-1">
              <Activity className="mr-1 h-3 w-3" />
              基于实时采集指标
            </div>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-background to-muted/50 border-primary/20">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">当前活跃 Pod</CardTitle>
            <TrendingDown className="h-4 w-4 text-green-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalPods}</div>
            <p className="text-xs text-muted-foreground mt-1">
              实时追踪中的 GPU 任务镜像
            </p>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-background to-muted/50 border-primary/20 ring-1 ring-primary/30">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">AI 诊断状态</CardTitle>
            <BrainCircuit className="h-4 w-4 text-purple-500 animate-pulse" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">已就绪</div>
            <p className="text-xs text-muted-foreground mt-1 text-purple-400">
              数据采集已闭环，等待聚合分析
            </p>
          </CardContent>
        </Card>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <Card className="col-span-2">
          <CardHeader>
            <CardTitle>活跃采集追踪列表 (Pod Traces)</CardTitle>
            <CardDescription>
              展示系统中所有正在运行的 GPU 容器及其关联池。
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>命名空间</TableHead>
                  <TableHead>Pod 名称</TableHead>
                  <TableHead>资源池</TableHead>
                  <TableHead>切分模式</TableHead>
                  <TableHead>实时算力</TableHead>
                  <TableHead>显存占用</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {traces.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center py-10 text-muted-foreground italic">
                      {loading ? "正在同步采集数据..." : "暂未发现活跃 GPU Pod 追踪信息。"}
                    </TableCell>
                  </TableRow>
                ) : (
                  traces.map((trace) => (
                    <TableRow key={`${trace.namespace}/${trace.pod_name}`}>
                      <TableCell className="text-xs">{trace.namespace}</TableCell>
                      <TableCell className="font-medium truncate max-w-[200px]">
                        {trace.pod_name}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="border-primary/30 text-primary">
                          {trace.pool_id}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <Badge variant="secondary">{trace.slicing_mode}</Badge>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2 font-mono text-xs text-green-500">
                          <Activity className="h-3 w-3" />
                          {trace.metrics?.gpu_util_avg?.toFixed(1) || "0.0"}%
                        </div>
                      </TableCell>
                      <TableCell className="text-xs font-mono">
                        {trace.metrics ? `${(trace.metrics.mem_used_bytes / 1024 / 1024 / 1024).toFixed(1)} GB` : "0 GB"}
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
