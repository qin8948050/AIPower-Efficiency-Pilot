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
  AreaChart,
  Area
} from "recharts";
import { DailyBillingSnapshot } from "@/lib/types";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Separator } from "@/components/ui/separator";
import { 
  Users, 
  TrendingUp, 
  Wallet, 
  PieChart as PieIcon, 
  BarChart3, 
  ArrowUpRight, 
  Target,
  Info,
  Activity,
  CalendarDays
} from "lucide-react";

const COLORS = ["#6366f1", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#06b6d4"];

interface TeamStats {
  name: string;
  cost: number;
  avgUtil: number;
  podCount: number;
}

export default function TeamsPage() {
  const [snapshots, setSnapshots] = useState<DailyBillingSnapshot[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedTeam, setSelectedTeam] = useState<string>("all");
  const [granularity, setGranularity] = useState<string>("day");

  useEffect(() => {
    fetch("http://localhost:8080/api/v2/billing/daily")
      .then((res) => res.json())
      .then((data) => {
        setSnapshots(data || []);
        setLoading(false);
      });
  }, []);

  // 1. 基础维度聚合 (用于概览)
  const teamMap = new Map<string, TeamStats>();
  snapshots.forEach((s) => {
    const team = s.TeamLabel || "未打标业务";
    const current = teamMap.get(team) || { name: team, cost: 0, avgUtil: 0, podCount: 0 };
    current.cost += s.TotalCost;
    current.podCount += s.PodSessionCount;
    current.avgUtil = (current.avgUtil === 0) ? s.AvgUtilP95 : (current.avgUtil + s.AvgUtilP95) / 2;
    teamMap.set(team, current);
  });

  const teamData = Array.from(teamMap.values()).sort((a, b) => b.cost - a.cost);
  const teams = teamData.map(t => t.name);

  // 2. 趋势数据处理 (支持 天、周、月)
  const getTrendData = () => {
    const dateMap = new Map<string, any>();
    
    snapshots.forEach((s) => {
      let dateKey = s.SnapshotDate;
      const dateObj = new Date(s.SnapshotDate);
      
      if (granularity === "week") {
        // 计算周一的日期作为 Key
        const day = dateObj.getDay();
        const diff = dateObj.getDate() - day + (day === 0 ? -6 : 1);
        const monday = new Date(dateObj.setDate(diff));
        dateKey = `W${monday.getMonth() + 1}-${monday.getDate()}`;
      } else if (granularity === "month") {
        dateKey = `${dateObj.getFullYear()}-${String(dateObj.getMonth() + 1).padStart(2, '0')}`;
      } else {
        dateKey = s.SnapshotDate.substring(5, 10); // MM-DD
      }

      const entry = dateMap.get(dateKey) || { date: dateKey };
      const teamKey = s.TeamLabel || "未打标业务";
      entry[teamKey] = (entry[teamKey] || 0) + s.TotalCost;
      dateMap.set(dateKey, entry);
    });

    return Array.from(dateMap.values()).sort((a, b) => a.date.localeCompare(b.date));
  };

  const trendData = getTrendData();

  // 3. 计算指标
  const totalCost = teamData.reduce((acc, curr) => acc + curr.cost, 0);
  const topTeam = teamData.length > 0 ? teamData[0] : null;
  const avgEfficiency = teamData.length > 0 ? teamData.reduce((acc, curr) => acc + curr.avgUtil, 0) / teamData.length : 0;

  return (
    <div className="flex-1 space-y-8 p-8 pt-6 bg-slate-50/30 min-h-screen">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 flex items-center gap-3">
            <Users className="h-8 w-8 text-emerald-600" />
            业务维度成本分析
          </h2>
          <p className="text-sm text-muted-foreground">
            精细化核算各业务线的算力支出，分析长期增长趋势与利用效能。
          </p>
        </div>
        <Badge variant="secondary" className="px-4 py-1.5 text-sm bg-white border-slate-200 shadow-sm font-bold text-slate-600">
          数据状态: 已实时同步
        </Badge>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-blue-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">总计核算成本</CardTitle>
            <Wallet className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-2xl font-bold text-slate-900">¥{totalCost.toLocaleString(undefined, {minimumFractionDigits: 2, maximumFractionDigits: 2})}</div>
            <p className="text-[10px] text-slate-400 mt-1">本期已聚合流水总额</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-rose-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">最高支出团队</CardTitle>
            <TrendingUp className="h-4 w-4 text-rose-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-xl font-bold text-slate-900 truncate">{topTeam?.name || "--"}</div>
            <p className="text-[10px] text-rose-500 font-medium mt-1 flex items-center gap-1">
              <ArrowUpRight size={10} /> 占比 {((topTeam?.cost || 0) / (totalCost || 1) * 100).toFixed(1)}%
            </p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-emerald-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">业务部平均能效</CardTitle>
            <Activity className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-2xl font-bold text-slate-900">{avgEfficiency.toFixed(1)}%</div>
            <p className="text-[10px] text-slate-400 mt-1">GPU 平均 P95 水位</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-amber-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">活跃业务单元</CardTitle>
            <Target className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-2xl font-bold text-slate-900">{teamData.length}</div>
            <p className="text-[10px] text-slate-400 mt-1">已打标且有消耗的团队</p>
          </CardContent>
        </Card>
      </div>

      {/* --- TREND CONSOLE: Filters & Chart in one card --- */}
      <Card className="border-slate-200 shadow-lg bg-white overflow-hidden ring-1 ring-slate-200/50">
        <CardHeader className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b bg-slate-50/50 py-4 px-6">
          <div className="flex items-center gap-3">
            <CalendarDays className="h-5 w-5 text-indigo-600" />
            <div>
              <CardTitle className="text-lg text-slate-800">算力费用支出趋势</CardTitle>
              <CardDescription className="text-sm text-muted-foreground">支持按业务线过滤，并按天/周/月进行趋势对标</CardDescription>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            {/* Team Selector */}
            <Select value={selectedTeam} onValueChange={setSelectedTeam}>
              <SelectTrigger className="w-[160px] h-9 bg-white border-slate-200 text-xs font-semibold">
                <Users className="mr-2 h-3.5 w-3.5 text-slate-400" />
                <SelectValue placeholder="筛选业务线" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部业务线</SelectItem>
                {teams.map(t => <SelectItem key={t} value={t}>{t}</SelectItem>)}
              </SelectContent>
            </Select>

            <Separator orientation="vertical" className="h-6 mx-1" />

            {/* Time Granularity Tabs */}
            <Tabs defaultValue="day" onValueChange={setGranularity} className="w-fit">
              <TabsList className="bg-slate-200/50 h-9">
                <TabsTrigger value="day" className="text-[11px] font-bold">天</TabsTrigger>
                <TabsTrigger value="week" className="text-[11px] font-bold">周</TabsTrigger>
                <TabsTrigger value="month" className="text-[11px] font-bold">月</TabsTrigger>
              </TabsList>
            </Tabs>
          </div>
        </CardHeader>
        
        <CardContent className="p-6">
          <div className="h-[350px] w-full">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={trendData}>
                <defs>
                  {teams.map((t, i) => (
                    <linearGradient key={`color-${t}`} id={`color-${t}`} x1="0" y1="0" x2="0" y2="1">
                      <stop offset="5%" stopColor={COLORS[i % COLORS.length]} stopOpacity={0.3}/>
                      <stop offset="95%" stopColor={COLORS[i % COLORS.length]} stopOpacity={0}/>
                    </linearGradient>
                  ))}
                </defs>
                <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
                <XAxis dataKey="date" fontSize={11} tickLine={false} axisLine={false} className="text-slate-400 font-medium" />
                <YAxis fontSize={11} tickLine={false} axisLine={false} tickFormatter={(v) => `¥${v}`} className="text-slate-400 font-mono" />
                <Tooltip 
                  contentStyle={{ borderRadius: '12px', border: 'none', boxShadow: '0 10px 15px -3px rgba(0,0,0,0.1)', padding: '12px' }}
                  cursor={{ stroke: '#e2e8f0', strokeWidth: 2 }}
                />
                <Legend iconType="circle" verticalAlign="top" align="right" wrapperStyle={{ paddingBottom: '20px', fontSize: '11px', fontWeight: 'bold' }} />
                {teams.map((t, i) => (
                  <Area 
                    key={t}
                    type="monotone" 
                    dataKey={t} 
                    stroke={COLORS[i % COLORS.length]} 
                    fillOpacity={1} 
                    fill={`url(#color-${t})`} 
                    strokeWidth={3}
                    stackId="1"
                    hide={selectedTeam !== "all" && selectedTeam !== t}
                    animationDuration={500}
                  />
                ))}
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </CardContent>
      </Card>

      <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-7">
        <Card className="lg:col-span-4 border-slate-200 shadow-sm bg-white">
          <CardHeader className="flex flex-row items-center gap-2 border-b bg-slate-50/50 py-4 px-6">
            <BarChart3 className="h-5 w-5 text-indigo-500" />
            <div>
              <CardTitle className="text-lg font-semibold text-slate-800">团队累计成本排行 (Total Cost)</CardTitle>
              <CardDescription className="text-sm text-muted-foreground">本期总支出金额降序</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="p-6">
            <div className="h-[300px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={teamData} layout="vertical" margin={{ left: 20, right: 40 }}>
                  <CartesianGrid strokeDasharray="3 3" horizontal={true} vertical={false} stroke="#f1f5f9" />
                  <XAxis type="number" hide />
                  <YAxis
                    dataKey="name"
                    type="category"
                    fontSize={12}
                    width={100}
                    tickLine={false}
                    axisLine={false}
                    className="font-bold text-slate-600"
                  />
                  <Tooltip
                    cursor={{ fill: '#f8fafc' }}
                    contentStyle={{ borderRadius: "8px", border: "none", boxShadow: "0 10px 15px -3px rgba(0,0,0,0.1)" }}
                    formatter={(value: number) => [`¥${value.toFixed(2)}`, "成本金额"]}
                  />
                  <Bar dataKey="cost" fill="#6366f1" radius={[0, 4, 4, 0]} barSize={24} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        <Card className="lg:col-span-3 border-slate-200 shadow-sm bg-white">
          <CardHeader className="flex flex-row items-center gap-2 border-b bg-slate-50/50 py-4 px-6">
            <PieIcon className="h-5 w-5 text-emerald-500" />
            <div>
              <CardTitle className="text-lg font-semibold text-slate-800">部门支出占比</CardTitle>
              <CardDescription className="text-sm text-muted-foreground">本期成本贡献度分析</CardDescription>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col items-center p-6">
            <div className="h-[250px] w-full">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie
                    data={teamData}
                    cx="50%"
                    cy="50%"
                    innerRadius={60}
                    outerRadius={80}
                    paddingAngle={5}
                    dataKey="cost"
                  >
                    {teamData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                    ))}
                  </Pie>
                  <Tooltip formatter={(v: number) => `¥${v.toFixed(2)}`} />
                  <Legend verticalAlign="bottom" height={36} iconType="circle" wrapperStyle={{fontSize: '11px', fontWeight: 'bold'}} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Team Ranking Table */}
      <Card className="border-slate-200 shadow-sm overflow-hidden bg-white">
        <CardHeader className="border-b bg-slate-50/50 py-4 px-6">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg text-slate-800">团队能效对标清单</CardTitle>
              <CardDescription className="text-xs">量化各部门资源使用的真实投入产出比</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader className="bg-slate-50/80">
              <TableRow>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase py-4 pl-6">业务部门 (Team Label)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">活跃 Pod 数量</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">总支出金额 (¥)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">平均能效水位 (P95)</TableHead>
                <TableHead className="text-right text-xs font-semibold text-slate-600 uppercase pr-6">治理评价</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {teamData.map((t) => (
                <TableRow key={t.name} className="hover:bg-slate-50 transition-colors border-b last:border-none">
                  <TableCell className="py-4 pl-6">
                    <div className="font-bold text-slate-800">{t.name}</div>
                    <div className="text-xs text-slate-400 mt-1">成本中心 ID: {t.name.toUpperCase()}-CC</div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-slate-700">{t.podCount}</span>
                      <div className="w-12 bg-slate-100 h-1 rounded-full">
                        <div className="bg-blue-400 h-full" style={{ width: `${Math.min((t.podCount / 50) * 100, 100)}%` }} />
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="font-mono font-bold text-slate-900 text-base">¥{t.cost.toFixed(2)}</div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className={`text-sm font-bold ${t.avgUtil > 60 ? "text-emerald-600" : t.avgUtil > 30 ? "text-blue-600" : "text-rose-500"}`}>
                        {t.avgUtil.toFixed(1)}%
                      </span>
                      <Activity className={`h-3 w-3 ${t.avgUtil > 60 ? "text-emerald-400" : "text-slate-300"}`} />
                    </div>
                  </TableCell>
                  <TableCell className="text-right pr-6">
                    <Badge 
                      variant={t.avgUtil > 60 ? "default" : t.avgUtil > 30 ? "secondary" : "destructive"}
                      className={t.avgUtil > 60 ? "bg-emerald-500" : ""}
                    >
                      {t.avgUtil > 60 ? "A 级能效" : t.avgUtil > 30 ? "B 级能效" : "C 级待优化"}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Help Insight */}
      <div className="flex items-start gap-3 p-4 bg-blue-50 border border-blue-100 rounded-xl">
        <Info className="h-5 w-5 text-blue-500 mt-0.5" />
        <div className="text-sm text-muted-foreground leading-relaxed">
          <p className="font-bold text-base mb-1">分摊逻辑说明：</p>
          本页面基于 <strong>K8s Pod Label (app.kubernetes.io/team)</strong> 进行自动归属。
          如果发现大量任务处于“未打标业务”，请督促各业务部门在 Yaml 中补充团队标签，以实现精准的算力财务对账。
        </div>
      </div>
    </div>
  );
}
