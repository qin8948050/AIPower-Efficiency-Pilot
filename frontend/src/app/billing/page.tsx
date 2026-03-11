"use client";

import React, { useEffect, useState } from "react";
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
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  Legend,
} from "recharts";
import { Wallet, TrendingUp, CreditCard, ArrowUpRight, ArrowDownRight } from "lucide-react";
import { DailyBillingSnapshot, LifeTrace } from "@/lib/types";

const COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#6366f1", "#ec4899", "#8b5cf6"];

export default function BillingPage() {
  const [dailySnapshots, setDailySnapshots] = useState<DailyBillingSnapshot[]>([]);
  const [transactions, setTransactions] = useState<LifeTrace[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // 1. 获取日级账单聚合数据 (用于趋势图 & 分布图)
        const dailyRes = await fetch("http://localhost:8080/api/v2/billing/daily"); // Replace with env var in prod
        const dailyData: DailyBillingSnapshot[] = await dailyRes.json();
        setDailySnapshots(dailyData || []);

        // 2. 获取最近的会话账单明细 (用于列表)
        const sessionsRes = await fetch("http://localhost:8080/api/v2/billing/sessions"); // Default to yesterday/today
        const sessionsData: LifeTrace[] = await sessionsRes.json();
        setTransactions(sessionsData || []);
      } catch (error) {
        console.error("Failed to fetch billing data:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  // --- Data Processing for Charts ---

  // 1. 成本趋势 (按日期聚合)
  const costTrendMap = new Map<string, number>();
  dailySnapshots.forEach((s) => {
    const date = s.SnapshotDate.substring(5, 10); // "MM-DD"
    const current = costTrendMap.get(date) || 0;
    costTrendMap.set(date, current + s.TotalCost);
  });
  // Sort by date
  const sortedDates = Array.from(costTrendMap.keys()).sort();
  const costData = sortedDates.map((date) => ({
    name: date,
    gpu: costTrendMap.get(date) || 0,
    cpu: 0, // 暂未统计 CPU 成本
    mem: 0, // 暂未统计 内存 成本
  }));

  // 2. 部门/池子分摊 (按 PoolID 聚合)
  const poolCostMap = new Map<string, number>();
  dailySnapshots.forEach((s) => {
    const current = poolCostMap.get(s.PoolID) || 0;
    poolCostMap.set(s.PoolID, current + s.TotalCost);
  });
  const distributionData = Array.from(poolCostMap.entries()).map(([name, value], index) => ({
    name,
    value,
    color: COLORS[index % COLORS.length],
  }));

  // 3. 汇总指标
  const totalEstimatedCost = dailySnapshots.reduce((acc, curr) => acc + curr.TotalCost, 0);
  const totalSessions = transactions.length;

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <CreditCard className="h-8 w-8 text-blue-600" />
          成本分摊中心
        </h2>
        <div className="flex items-center space-x-2">
          <Badge variant="outline" className="px-3 py-1 text-sm">
            数据来源: 实时聚合
          </Badge>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">总预估成本 (本期)</CardTitle>
            <Wallet className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">¥{totalEstimatedCost.toFixed(2)}</div>
            <p className="text-xs text-muted-foreground flex items-center mt-1">
              基于已聚合的日级快照
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">GPU 算力成本</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">¥{totalEstimatedCost.toFixed(2)}</div>
            <p className="text-xs text-muted-foreground mt-1">占比 100% (当前仅计 GPU)</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">闲置资源损耗</CardTitle>
            <ArrowDownRight className="h-4 w-4 text-rose-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-rose-500">--</div>
            <p className="text-xs text-muted-foreground mt-1">需 Phase 3 AI 分析接入</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">近日会话数</CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalSessions}</div>
            <p className="text-xs text-muted-foreground mt-1">最近 24 小时活跃 Pod</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        {/* Cost Trend Chart */}
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>算力成本趋势 (Daily)</CardTitle>
            <CardDescription className="text-sm text-muted-foreground">最近日期的 GPU 算力支出分布</CardDescription>          </CardHeader>
          <CardContent className="pl-2">
            <div className="h-[350px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={costData}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="name" fontSize={12} tickLine={false} axisLine={false} />
                  <YAxis fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `¥${value}`} />
                  <Tooltip
                    cursor={{ fill: 'rgba(0,0,0,0.05)' }}
                    contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
                  />
                  <Legend iconType="circle" />
                  <Bar dataKey="gpu" name="GPU 成本" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* Distribution Chart */}
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>资源池成本分担</CardTitle>
            <CardDescription className="text-sm text-muted-foreground">各资源池累计支出占比</CardDescription>          </CardHeader>
          <CardContent>
            <div className="h-[300px] w-full flex items-center justify-center">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={distributionData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="value"
                  >
                    {distributionData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend verticalAlign="bottom" height={36} iconType="circle" />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Transaction Table */}
      <Card>
        <CardHeader>
          <CardTitle>最近结算记录</CardTitle>
          <CardDescription className="text-sm text-muted-foreground">Pod 会话级账单明细</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">Pod Name</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">Namespace</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">资源池</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">切片模式</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">开始时间</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">费用 (¥)</TableHead>
                <TableHead className="text-right text-xs font-semibold text-slate-600 uppercase">状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((tx) => (
                <TableRow key={tx.ID}>
                  <TableCell className="font-medium">{tx.PodName}</TableCell>
                  <TableCell>{tx.Namespace}</TableCell>
                  <TableCell>{tx.PoolID}</TableCell>
                  <TableCell>{tx.SlicingMode}</TableCell>
                  <TableCell>{new Date(tx.StartTime).toLocaleString()}</TableCell>
                  <TableCell>¥{tx.CostAmount.toFixed(4)}</TableCell>
                  <TableCell className="text-right">
                    <Badge variant={tx.Status === "Settled" ? "outline" : "secondary"}>
                      {tx.Status === "Settled" ? "已结算" : tx.Status === "Auditing" ? "审计中" : "运行中"}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}

