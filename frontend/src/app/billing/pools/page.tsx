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
import { Box, BarChart3, Coins, Zap } from "lucide-react";

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
          current.avgUtilP95 = (current.avgUtilP95 + s.AvgUtilP95) / 2; // 简化平均
          current.maxMemGiB = Math.max(current.maxMemGiB, s.MaxMemGiB);
          current.sessionCount += s.PodSessionCount;
          map.set(s.PoolID, current);
        });

        const result = Array.from(map.values()).map(p => ({
          ...p,
          unitCost: p.totalCost / (p.avgUtilP95 || 1) // 算出每 1% 利用率花了几块钱
        }));
        setPools(result.sort((a, b) => b.totalCost - a.totalCost));
      });
  }, []);

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">资源池效能量化</h2>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {pools.slice(0, 3).map((p) => (
          <Card key={p.poolID}>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium flex items-center">
                <Box className="mr-2 h-4 w-4 text-blue-500" />
                {p.poolID}
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">¥{p.totalCost.toFixed(2)}</div>
              <div className="flex items-center mt-2 space-x-4 text-xs text-muted-foreground">
                <span className="flex items-center">
                  <BarChart3 className="mr-1 h-3 w-3" /> P95: {p.avgUtilP95.toFixed(1)}%
                </span>
                <span className="flex items-center">
                  <Zap className="mr-1 h-3 w-3 text-amber-500" /> 单位成本: ¥{p.unitCost.toFixed(2)}
                </span>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card>
        <CardHeader>
          <CardTitle>资源池 ROI 排名</CardTitle>
          <CardDescription>
            量化每个池子的算力性价比与显存使用效率
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>资源池</TableHead>
                <TableHead>总支出 (本期)</TableHead>
                <TableHead>算力性价比 (单位成本)</TableHead>
                <TableHead>GPU 利用率 (P95)</TableHead>
                <TableHead>显存使用峰值</TableHead>
                <TableHead className="text-right">健康度</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((p) => (
                <TableRow key={p.poolID}>
                  <TableCell className="font-medium">{p.poolID}</TableCell>
                  <TableCell>¥{p.totalCost.toFixed(2)}</TableCell>
                  <TableCell>
                    <div className="flex flex-col">
                      <span className="text-sm font-mono text-emerald-600">¥{p.unitCost.toFixed(2)} / 1%</span>
                      <span className="text-[10px] text-muted-foreground">每单位算力支出</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="w-[120px] space-y-1">
                      <div className="flex justify-between text-[10px]">
                        <span>P95 水位</span>
                        <span>{p.avgUtilP95.toFixed(1)}%</span>
                      </div>
                      <Progress value={p.avgUtilP95} className="h-1.5" />
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="font-medium">{p.maxMemGiB.toFixed(1)} GiB</div>
                  </TableCell>
                  <TableCell className="text-right">
                    <Badge 
                      variant={p.avgUtilP95 > 60 ? "default" : p.avgUtilP95 > 30 ? "secondary" : "destructive"}
                      className={p.avgUtilP95 > 60 ? "bg-emerald-500" : ""}
                    >
                      {p.avgUtilP95 > 60 ? "高效" : p.avgUtilP95 > 30 ? "一般" : "低效"}
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
