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
import { Progress } from "@/components/ui/progress";
import { DailyBillingSnapshot } from "@/lib/types";
import { 
  Box, 
  BarChart3, 
  Zap, 
  TrendingUp, 
  Activity, 
  Trophy, 
  ArrowUpRight, 
  Microchip,
  Wallet
} from "lucide-react";

interface PoolEfficiency {
  poolID: string;
  totalCost: number;
  avgUtilP95: number;
  maxMemGiB: number;
  sessionCount: number;
  unitCost: number; // 单位算力成本: TotalCost / AvgUtil
}

export default function PoolEfficiencyPage() {
  const [pools, setPools] = useState<PoolEfficiency[]>([]);

  useEffect(() => {
    fetch("http://localhost:8080/api/v2/billing/daily")
      .then((res) => res.json())
      .then((data: DailyBillingSnapshot[]) => {
        const map = new Map<string, PoolEfficiency>();
        data.forEach((s) => {
          const current = map.get(s.PoolID) || {
            poolID: s.PoolID,
            totalCost: 0,
            avgUtilP95: 0,
            maxMemGiB: 0,
            sessionCount: 0,
            unitCost: 0,
          };
          current.totalCost += s.TotalCost;
          // 简单的加权平均逻辑 (实际生产中应按天数加权)
          current.avgUtilP95 = (current.avgUtilP95 === 0) ? s.AvgUtilP95 : (current.avgUtilP95 + s.AvgUtilP95) / 2;
          current.maxMemGiB = Math.max(current.maxMemGiB, s.MaxMemGiB);
          current.sessionCount += s.PodSessionCount;
          map.set(s.PoolID, current);
        });

        const result = Array.from(map.values()).map(p => ({
          ...p,
          unitCost: p.totalCost / (p.avgUtilP95 || 1)
        }));
        // 按单位成本升序排列 (越低越高效)
        setPools(result.sort((a, b) => a.unitCost - b.unitCost));
      });
  }, []);

  const topEfficiency = pools.length > 0 ? pools[0] : null;

  return (
    <div className="flex-1 space-y-6 p-8 pt-6 bg-slate-50/30 min-h-screen">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 flex items-center gap-2">
            <BarChart3 className="h-8 w-8 text-indigo-500" />
            资源池效能量化
          </h2>
          <p className="text-sm text-muted-foreground flex items-center gap-1">
            <Activity className="h-3 w-3" />
            衡量每一分算力投入的真实产出，评估硬件采购的经济效益
          </p>
        </div>
        <Badge variant="outline" className="px-4 py-1.5 text-sm bg-white border-indigo-100 text-indigo-700 shadow-sm font-medium">
          核心指标: 单位算力成本 (¥/1%)
        </Badge>
      </div>

      {/* Hero ROI Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        {pools.slice(0, 3).map((p, idx) => (
          <Card key={p.poolID} className={`relative overflow-hidden border-none shadow-lg text-white ${
            idx === 0 ? "bg-gradient-to-br from-indigo-600 to-blue-700" : 
            idx === 1 ? "bg-gradient-to-br from-blue-500 to-cyan-600" : 
            "bg-gradient-to-br from-slate-700 to-slate-800"
          }`}>
            <div className="absolute right-[-10px] top-[-10px] opacity-10">
              {idx === 0 ? <Trophy size={120} /> : <Box size={120} />}
            </div>
            <CardHeader className="pb-2">
              <div className="flex justify-between items-start">
                <CardTitle className="text-sm font-medium flex items-center gap-2 opacity-90">
                  {idx === 0 && <Trophy className="h-4 w-4 text-amber-300" />}
                  {p.poolID}
                </CardTitle>
                <Badge className="bg-white/20 hover:bg-white/30 border-none text-white text-[10px]">
                  RANK #{idx + 1}
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col">
                <span className="text-xs opacity-70 mb-1">单位算力成本 (ROI)</span>
                <div className="text-3xl font-bold flex items-baseline gap-1">
                  ¥{p.unitCost.toFixed(2)}
                  <span className="text-xs font-normal opacity-80">/ 1% Util</span>
                </div>
              </div>
              <div className="mt-4 grid grid-cols-2 gap-2 pt-4 border-t border-white/10 text-[11px]">
                <div className="flex items-center gap-1.5">
                  <Activity className="h-3 w-3 opacity-70" />
                  P95: {p.avgUtilP95.toFixed(1)}%
                </div>
                <div className="flex items-center gap-1.5">
                  <Wallet className="h-3 w-3 opacity-70" />
                  总支出: ¥{Math.round(p.totalCost)}
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="border-slate-200 shadow-sm overflow-hidden bg-white">
        <CardHeader className="border-b bg-slate-50/50">
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg flex items-center text-slate-800">
                <Microchip className="mr-2 h-5 w-5 text-blue-600" />
                全量池效能 ROI 排名
              </CardTitle>
              <CardDescription className="text-sm text-muted-foreground">
                单位成本越低说明该池子承载的任务与硬件能力匹配度越高
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader className="bg-slate-50/80">
              <TableRow>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase py-4 pl-6">资源池标识</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">总支出 (本周期)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">算力性价比 (单位成本)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">利用流水位 (P95)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">显存压力峰值</TableHead>
                <TableHead className="text-right text-xs font-semibold text-slate-600 uppercase pr-6">效能健康度</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((p) => (
                <TableRow key={p.poolID} className="hover:bg-slate-50 transition-colors border-b last:border-none">
                  <TableCell className="py-4 pl-6">
                    <div className="font-bold text-slate-800">{p.poolID}</div>
                    <div className="text-[10px] text-slate-400 mt-1 flex items-center gap-1">
                      <Box className="h-3 w-3" />
                      Session Count: {p.sessionCount}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="font-semibold text-slate-700 font-mono">¥{p.totalCost.toFixed(2)}</div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <div className="flex items-center gap-1.5">
                        <span className="text-sm font-bold text-emerald-600 font-mono">¥{p.unitCost.toFixed(2)}</span>
                        <ArrowUpRight className="h-3 w-3 text-emerald-500" />
                      </div>
                      <span className="text-[10px] text-slate-400">每产生 1% 算力的平均投入</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="w-[160px] space-y-1.5">
                      <div className="flex justify-between items-center text-[10px] font-medium text-slate-500">
                        <span>P95 水位</span>
                        <span>{p.avgUtilP95.toFixed(1)}%</span>
                      </div>
                      <Progress 
                        value={p.avgUtilP95} 
                        className={`h-1.5 ${p.avgUtilP95 > 70 ? "[&>div]:bg-emerald-500" : p.avgUtilP95 > 30 ? "[&>div]:bg-blue-500" : "[&>div]:bg-rose-500"}`} 
                      />
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-col gap-0.5">
                      <span className="text-sm font-bold text-slate-700">{p.maxMemGiB.toFixed(1)} <span className="text-[10px] font-normal text-slate-400 italic">GiB</span></span>
                      <div className="text-[10px] text-slate-400">全周期最高水位</div>
                    </div>
                  </TableCell>
                  <TableCell className="text-right pr-6">
                    <Badge 
                      variant={p.avgUtilP95 > 60 ? "default" : p.avgUtilP95 > 30 ? "secondary" : "destructive"}
                      className={`px-3 py-1 ${p.avgUtilP95 > 60 ? "bg-emerald-500 hover:bg-emerald-600" : ""}`}
                    >
                      {p.avgUtilP95 > 60 ? "高效产出" : p.avgUtilP95 > 30 ? "能效一般" : "低性价比"}
                    </Badge>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Insight Alert */}
      <div className="p-4 rounded-xl bg-indigo-50 border border-indigo-100 flex items-start gap-4">
        <div className="p-2 bg-indigo-100 rounded-lg text-indigo-600">
          <Zap size={20} />
        </div>
        <div>
          <h4 className="text-base font-bold text-indigo-900">效能提示 (ROI Insights)</h4>
          <p className="text-sm text-muted-foreground mt-1 leading-relaxed">
            当前 <strong>{topEfficiency?.poolID}</strong> 表现出最高的性价比。如果其他相同硬件型号的资源池（如 A100）单位成本显著更高，请关注其是否存在大量“显存长期占用但算力闲置”的任务。
          </p>
        </div>
      </div>
    </div>
  );
}

