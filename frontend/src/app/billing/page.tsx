"use client";

import React from "react";
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

const costData = [
  { name: "1月", cpu: 4500, gpu: 12000, mem: 2100 },
  { name: "2月", cpu: 5200, gpu: 15000, mem: 2400 },
  { name: "3月", cpu: 4800, gpu: 18000, mem: 2200 },
  { name: "4月", cpu: 6100, gpu: 22000, mem: 3100 },
  { name: "5月", cpu: 5900, gpu: 21000, mem: 2800 },
  { name: "6月", cpu: 7200, gpu: 28000, mem: 3500 },
];

const distributionData = [
  { name: "训练集群-A", value: 45, color: "#3b82f6" },
  { name: "推理集群-B", value: 30, color: "#10b981" },
  { name: "开发环境-C", value: 15, color: "#f59e0b" },
  { name: "其他项目", value: 10, color: "#6366f1" },
];

const transactions = [
  { id: "TX-9021", date: "2024-03-08", dept: "AI 实验室", amount: "¥12,450.00", status: "已结算", type: "GPU 训练" },
  { id: "TX-9022", date: "2024-03-07", dept: "内容安全部", amount: "¥8,200.00", status: "处理中", type: "模型推理" },
  { id: "TX-9023", date: "2024-03-05", dept: "搜索算法组", amount: "¥15,600.00", status: "已结算", type: "混合算力" },
  { id: "TX-9024", date: "2024-03-04", dept: "平台架构层", amount: "¥3,500.00", status: "已结算", type: "基础资源" },
];

export default function BillingPage() {
  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">成本分摊中心</h2>
        <div className="flex items-center space-x-2">
          <Badge variant="outline" className="px-3 py-1 text-sm">
            结算周期: 2024 Q1
          </Badge>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">本月预估总额</CardTitle>
            <Wallet className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">¥38,700.00</div>
            <p className="text-xs text-muted-foreground flex items-center mt-1">
              <span className="text-emerald-500 flex items-center font-medium mr-1">
                <ArrowUpRight className="h-3 w-3 mr-0.5" /> +12.5%
              </span>
              较上月同期
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">GPU 核心成本</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">¥28,000.00</div>
            <p className="text-xs text-muted-foreground mt-1">占比 72.3%</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">闲置资源损耗</CardTitle>
            <ArrowDownRight className="h-4 w-4 text-rose-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-rose-500">¥1,250.00</div>
            <p className="text-xs text-muted-foreground mt-1">优化建议: 缩容 15%</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">待结算账单</CardTitle>
            <CreditCard className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">¥12,450.00</div>
            <p className="text-xs text-muted-foreground mt-1">包含 2 个待处理项目</p>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        {/* Cost Trend Chart */}
        <Card className="col-span-4">
          <CardHeader>
            <CardTitle>算力成本趋势</CardTitle>
            <CardDescription>最近 6 个月的 CPU/GPU/内存 支出分布</CardDescription>
          </CardHeader>
          <CardContent className="pl-2">
            <div className="h-[350px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={costData}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} />
                  <XAxis dataKey="name" fontSize={12} tickLine={false} axisLine={false} />
                  <YAxis fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `¥${value / 1000}k`} />
                  <Tooltip
                    cursor={{ fill: 'rgba(0,0,0,0.05)' }}
                    contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }}
                  />
                  <Legend iconType="circle" />
                  <Bar dataKey="gpu" name="GPU 成本" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="cpu" name="CPU 成本" fill="#10b981" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="mem" name="内存 成本" fill="#f59e0b" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        {/* Distribution Chart */}
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>部门成本分担</CardTitle>
            <CardDescription>当前季度各资源池支出占比</CardDescription>
          </CardHeader>
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
            <div className="mt-4 space-y-4">
              <div className="flex items-center text-sm">
                <div className="flex-1 font-medium">最高支出项目: 训练集群-A</div>
                <div className="text-muted-foreground">¥17.4k</div>
              </div>
              <div className="w-full bg-secondary h-2 rounded-full overflow-hidden">
                <div className="bg-blue-500 h-full w-[45%]" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Transaction Table */}
      <Card>
        <CardHeader>
          <CardTitle>最近结算记录</CardTitle>
          <CardDescription>查看详细的资源使用结算清单</CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>账单 ID</TableHead>
                <TableHead>使用部门</TableHead>
                <TableHead>资源类型</TableHead>
                <TableHead>产生日期</TableHead>
                <TableHead>金额</TableHead>
                <TableHead className="text-right">状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {transactions.map((tx) => (
                <TableRow key={tx.id}>
                  <TableCell className="font-medium">{tx.id}</TableCell>
                  <TableCell>{tx.dept}</TableCell>
                  <TableCell>{tx.type}</TableCell>
                  <TableCell>{tx.date}</TableCell>
                  <TableCell>{tx.amount}</TableCell>
                  <TableCell className="text-right">
                    <Badge variant={tx.status === "已结算" ? "outline" : "secondary"}>
                      {tx.status}
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
