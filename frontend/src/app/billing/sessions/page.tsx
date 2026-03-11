"use client";

import React, { useEffect, useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { LifeTrace } from "@/lib/types";
import {
  Search,
  AlertCircle,
  Clock,
  Zap,
  Activity,
  Users,
  Filter,
  CheckCircle2,
  BarChart4,
  ArrowDownCircle,
  Terminal,
  ChevronDown,
  ChevronUp,
  RotateCcw,
  ListFilter
} from "lucide-react";

const formatDuration = (start: string, end: string | null) => {
  const startTime = new Date(start).getTime();
  const endTime = end ? new Date(end).getTime() : new Date().getTime();
  const diffMs = endTime - startTime;
  const mins = Math.floor(diffMs / 60000);
  const hours = Math.floor(mins / 60);
  if (hours > 0) return `${hours}h ${mins % 60}m`;
  return `${mins}m`;
};

export default function SessionsPage() {
  const [sessions, setSessions] = useState<LifeTrace[]>([]);
  const [searchQuery, setSearchQuery] = useState("");
  const [loading, setLoading] = useState(true);
  const [showFilters, setShowFilters] = useState(false);

  const [filters, setFilters] = useState({
    namespace: "all",
    poolID: "all",
    mode: "all",
    status: "all",
  });

  useEffect(() => {
    fetch("http://localhost:8080/api/v2/billing/sessions")
      .then((res) => res.json())
      .then((data) => {
        setSessions(data || []);
        setLoading(false);
      });
  }, []);

  const filtered = sessions.filter((s) => {
    const matchesSearch =
      s.PodName.toLowerCase().includes(searchQuery.toLowerCase()) ||
      s.Namespace.toLowerCase().includes(searchQuery.toLowerCase()) ||
      (s.TeamLabel && s.TeamLabel.toLowerCase().includes(searchQuery.toLowerCase()));
    const matchesNamespace = filters.namespace === "all" || s.Namespace === filters.namespace;
    const matchesPool = filters.poolID === "all" || s.PoolID === filters.poolID;
    const matchesMode = filters.mode === "all" || s.SlicingMode === filters.mode;
    const matchesStatus = filters.status === "all" || s.Status === filters.status || (filters.status === "completed" && s.Status === "Settled") || (filters.status === "running" && s.Status === "Running");
    return matchesSearch && matchesNamespace && matchesPool && matchesMode && matchesStatus;
  });

  const namespaces = Array.from(new Set(sessions.map(s => s.Namespace))).sort();
  const pools = Array.from(new Set(sessions.map(s => s.PoolID))).sort();
  const modes = Array.from(new Set(sessions.map(s => s.SlicingMode))).sort();
  const hasActiveFilters = filters.namespace !== "all" || filters.poolID !== "all" || filters.mode !== "all" || filters.status !== "all";

  const lowEfficiencyCount = filtered.filter(s => s.GPUUtilAvg < 30 && s.Status === "Settled").length;
  const avgEfficiency = filtered.length > 0 ? filtered.reduce((acc, curr) => acc + curr.GPUUtilAvg, 0) / filtered.length : 0;

  return (
    <div className="flex-1 space-y-10 p-8 pt-6 bg-slate-50/30 min-h-screen">
      {/* --- Section 1: Page Header --- */}
      <div className="flex flex-col gap-2">
        <h2 className="text-3xl font-bold tracking-tight text-slate-900 flex items-center gap-3">
          <Terminal className="h-8 w-8 text-indigo-600" />
          Pod 会话级效能审计
        </h2>
        <p className="text-sm text-muted-foreground max-w-2xl">
          深度追踪 GPU 任务生命周期，系统基于采集指标自动分析算力性价比，为您提供精准的治理决策依据。
        </p>
      </div>

      {/* --- Section 2: Summary Stats (Breathing Space Added) --- */}
      <div className="grid gap-6 md:grid-cols-4">
        <Card className="bg-white border-slate-200 shadow-sm border-t-4 border-t-indigo-500 transition-transform hover:translate-y-[-2px]">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">待治理低效任务</CardTitle>
            <ArrowDownCircle className="h-4 w-4 text-rose-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-3xl font-bold text-slate-900">{lowEfficiencyCount}</div>
            <p className="text-[10px] text-slate-400 mt-1">平均利用率 &lt; 30%</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-t-4 border-t-emerald-500 transition-transform hover:translate-y-[-2px]">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">当前平均能效</CardTitle>
            <BarChart4 className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-3xl font-bold text-slate-900">{avgEfficiency.toFixed(1)}%</div>
            <div className="w-full bg-slate-100 h-1 rounded-full mt-2 overflow-hidden">
              <div className="bg-emerald-500 h-full" style={{ width: `${avgEfficiency}%` }} />
            </div>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-t-4 border-t-blue-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">审计覆盖度</CardTitle>
            <CheckCircle2 className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-3xl font-bold text-slate-900">100%</div>
            <p className="text-[10px] text-emerald-600 font-medium mt-1">资产已全量同步</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-t-4 border-t-amber-500">
          <CardHeader className="py-4 px-5 flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-xs font-bold text-slate-400 uppercase">健康评价</CardTitle>
            <Activity className="h-4 w-4 text-amber-500" />
          </CardHeader>
          <CardContent className="px-5 pb-5">
            <div className="text-3xl font-bold text-amber-600 font-mono">{avgEfficiency > 50 ? "A-" : "B"}</div>
            <p className="text-[10px] text-slate-400 mt-1">基于当前筛选结果</p>
          </CardContent>
        </Card>
      </div>

      {/* --- Section 3: Primary Workspace (Unified but Spaced) --- */}
      <div className="space-y-6">
        {/* Unified Search & Filter Console */}
        <Card className="border-slate-200 shadow-md bg-white overflow-hidden ring-1 ring-slate-200/50">
          <div className="p-5 bg-slate-50/50 border-b space-y-5">
            <div className="flex items-center gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
                <Input
                  placeholder="快速搜索 Pod 名称、Namespace、业务团队..."
                  className="pl-10 h-12 bg-white border-slate-200 focus:ring-4 focus:ring-blue-500/10 transition-all text-base shadow-sm"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              <Button
                variant={showFilters ? "default" : "outline"}
                className={`h-12 px-6 gap-2 font-bold shadow-sm transition-all ${!showFilters && hasActiveFilters ? "border-blue-500 text-blue-600 bg-blue-50" : ""}`}
                onClick={() => setShowFilters(!showFilters)}
              >
                <ListFilter className="h-4 w-4" />
                {showFilters ? "隐藏筛选" : "高级精准筛选"}
                {showFilters ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
              </Button>
              {hasActiveFilters && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-12 w-12 text-slate-400 hover:text-rose-500 hover:bg-rose-50 border border-dashed border-slate-200"
                  onClick={() => setFilters({ namespace: "all", poolID: "all", mode: "all", status: "all" })}
                >
                  <RotateCcw size={20} />
                </Button>
              )}
            </div>

            {/* Embedded Filter Grid */}
            {showFilters && (
              <div className="grid grid-cols-1 md:grid-cols-4 gap-6 p-6 bg-white rounded-xl border border-slate-200 shadow-inner animate-in fade-in slide-in-from-top-2 duration-300">
                <div className="space-y-2">
                  <label className="text-xs font-bold text-slate-500 uppercase tracking-widest px-1">命名空间</label>
                  <Select value={filters.namespace} onValueChange={(val: string) => setFilters({ ...filters, namespace: val })}>
                    <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部命名空间</SelectItem>
                      {namespaces.map(ns => <SelectItem key={ns} value={ns}>{ns}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-bold text-slate-500 uppercase tracking-widest px-1">算力池</label>
                  <Select value={filters.poolID} onValueChange={(val: string) => setFilters({ ...filters, poolID: val })}>
                    <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部算力资源池</SelectItem>
                      {pools.map(p => <SelectItem key={p} value={p}>{p}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-bold text-slate-500 uppercase tracking-widest px-1">切分模式</label>
                  <Select value={filters.mode} onValueChange={(val: string) => setFilters({ ...filters, mode: val })}>
                    <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">全部虚拟化模式</SelectItem>
                      {modes.map(m => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-bold text-slate-500 uppercase tracking-widest px-1">结算状态</label>
                  <Select value={filters.status} onValueChange={(val: string) => setFilters({ ...filters, status: val })}>
                    <SelectTrigger className="h-10"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">不限状态</SelectItem>
                      <SelectItem value="Settled">已结算</SelectItem>
                      <SelectItem value="Auditing">审计中</SelectItem>
                      <SelectItem value="Running">运行中</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            )}
          </div>

          {/* Table Area with distinct result header */}
          <div className="p-0 bg-white">
            <div className="px-6 py-4 flex items-center justify-between border-b bg-slate-50/30">
              <div className="flex items-center gap-2">
                <Badge variant="outline" className="bg-white text-indigo-600 border-indigo-100 font-bold px-3 py-1">
                  {filtered.length} / {sessions.length}
                </Badge>
                <span className="text-xs font-semibold text-slate-500 uppercase tracking-tight">条记录匹配当前视图</span>
              </div>
              <div className="text-[10px] text-slate-400 font-medium">排序依据: 结束时间 (倒序)</div>
            </div>

            <Table>
              <TableHeader className="bg-slate-50/50">
                <TableRow>
                  <TableHead className="py-4 pl-6 text-xs font-semibold text-slate-600 uppercase">Pod 信息 / 业务</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase">资源配置</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase">运行时长</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase">利用率 (Avg/Peak)</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase">显存 & 功耗</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase text-right">计费金额</TableHead>
                  <TableHead className="text-xs font-semibold text-slate-600 uppercase text-right pr-6">状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={7} className="h-80 text-center">
                      <div className="flex flex-col items-center justify-center gap-3 text-slate-400">
                        <div className="p-4 bg-slate-100 rounded-full">
                          <Search size={40} className="opacity-30" />
                        </div>
                        <p className="text-sm font-medium">未找到符合条件的审计记录</p>
                        <Button variant="outline" size="sm" onClick={() => {
                          setSearchQuery("");
                          setFilters({ namespace: "all", poolID: "all", mode: "all", status: "all" });
                        }}>清空所有搜索条件</Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filtered.map((s) => (
                    <TableRow key={s.ID} className="hover:bg-blue-50/40 transition-colors border-b last:border-none group/row cursor-default">
                      <TableCell className="py-5 pl-6">
                        <div className="font-bold text-slate-800 group-hover/row:text-blue-600 transition-colors">{s.PodName}</div>
                        <div className="flex items-center gap-2 mt-2">
                          <span className="text-xs bg-slate-100 text-slate-500 px-1.5 py-0.5 rounded font-bold border border-slate-200">NS: {s.Namespace}</span>
                          <div className="flex items-center gap-1 text-xs text-slate-400 font-semibold">
                            <Users className="h-3 w-3" />
                            {s.TeamLabel || "未归属"}
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="text-xs font-bold text-slate-700">{s.PoolID}</div>
                        <div className="mt-1 flex items-center gap-1.5">
                          <div className="w-1.5 h-1.5 rounded-full bg-blue-500 shadow-sm shadow-blue-200" />
                          <span className="text-xs text-slate-400 font-mono tracking-tighter uppercase">{s.SlicingMode} Mode</span>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center text-xs font-bold text-slate-600 bg-slate-100/80 border border-slate-200 px-2.5 py-1 rounded-full shadow-sm w-fit">
                          <Clock className="mr-1.5 h-3 w-3 text-indigo-500" />
                          {formatDuration(s.StartTime, s.EndTime)}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="space-y-2">
                          <div className="flex items-center justify-between w-32">
                            <span className={`text-sm font-black ${s.GPUUtilAvg < 30 ? "text-rose-500" : "text-emerald-600"}`}>
                              {s.GPUUtilAvg.toFixed(1)}% <span className="font-normal text-[10px] opacity-60">Avg</span>
                            </span>
                          </div>
                          <div className="w-32 bg-slate-100 h-1.5 rounded-full overflow-hidden flex shadow-inner border border-slate-200/50">
                            <div className={`${s.GPUUtilAvg < 30 ? "bg-rose-400 shadow-sm shadow-rose-200" : "bg-emerald-400 shadow-sm shadow-emerald-200"} h-full transition-all duration-500`} style={{ width: `${s.GPUUtilAvg}%` }} />
                            <div className="bg-slate-300 h-full opacity-20" style={{ width: `${s.GPUUtilMax - s.GPUUtilAvg}%` }} />
                          </div>
                          {s.GPUUtilAvg < 30 && s.Status === "Settled" && (
                            <div className="flex items-center text-[9px] text-rose-500 font-black uppercase tracking-tighter gap-1">
                              <AlertCircle size={10} strokeWidth={3} /> Low Performance
                            </div>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-col gap-1.5">
                          <div className="text-[11px] font-bold text-slate-700 leading-none">{s.MemUsedMax} <span className="font-medium text-slate-400">MiB</span></div>
                          <div className="flex items-center gap-1 text-[10px] text-slate-400 font-medium">
                            <Zap className="h-3 w-3 text-amber-500" />
                            {s.PowerUsageAvg.toFixed(0)}W Power
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="font-mono font-black text-lg text-slate-900 tracking-tighter">
                          ¥{s.CostAmount.toFixed(4)}
                        </div>
                      </TableCell>
                      <TableCell className="text-right pr-6">
                        <Badge
                          variant={s.Status === "Settled" ? "outline" : "default"}
                          className={s.Status === "Settled"
                            ? "border-emerald-200 text-emerald-700 bg-emerald-50 text-[10px] font-black px-2.5 py-1 uppercase"
                            : s.Status === "Auditing"
                            ? "bg-amber-500 text-white text-[10px] font-black px-2.5 py-1 uppercase"
                            : "bg-indigo-600 text-white animate-pulse text-[10px] font-black px-2.5 py-1 uppercase shadow-lg shadow-indigo-200"
                          }
                        >
                          {s.Status === "Settled" ? "已结算" : s.Status === "Auditing" ? "审计中" : "运行中"}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </Card>
      </div>
    </div>
  );
}
