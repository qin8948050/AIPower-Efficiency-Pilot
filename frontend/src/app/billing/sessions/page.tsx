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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { LifeTrace } from "@/lib/types";
import { Search, AlertCircle, Clock, Zap, Activity, Users } from "lucide-react";

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
  const [filter, setFilter] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("http://localhost:8080/api/v2/billing/sessions")
      .then((res) => res.json())
      .then((data) => {
        setSessions(data || []);
        setLoading(false);
      });
  }, []);

  const filtered = sessions.filter(
    (s) =>
      s.PodName.toLowerCase().includes(filter.toLowerCase()) ||
      s.Namespace.toLowerCase().includes(filter.toLowerCase()) ||
      (s.TeamLabel && s.TeamLabel.toLowerCase().includes(filter.toLowerCase()))
  );

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">Pod 会话级效能审计</h2>
        <Badge variant="outline" className="text-sm">
          共 {filtered.length} 条结算记录
        </Badge>
      </div>

      <div className="flex items-center space-x-2">
        <div className="relative flex-1">
          <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="搜索 Pod、命名空间 或 业务团队..."
            className="pl-8"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          />
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center">
            <Activity className="mr-2 h-5 w-5 text-blue-500" />
            资源使用与计费明细
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[200px]">Pod 信息 / 业务</TableHead>
                <TableHead>资源池 / 模式</TableHead>
                <TableHead>运行时长</TableHead>
                <TableHead>利用率 (Avg/Max)</TableHead>
                <TableHead>显存峰值</TableHead>
                <TableHead>平均功耗</TableHead>
                <TableHead>计费金额</TableHead>
                <TableHead className="text-right">结算状态</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((s) => (
                <TableRow key={s.ID} className="hover:bg-muted/50 transition-colors">
                  <TableCell>
                    <div className="font-medium truncate max-w-[180px]" title={s.PodName}>
                      {s.PodName}
                    </div>
                    <div className="flex flex-col gap-1 mt-1">
                      <div className="flex items-center gap-1 text-[10px] text-muted-foreground bg-secondary/50 px-1.5 py-0.5 rounded w-fit">
                        <span className="font-semibold">NS:</span> {s.Namespace}
                      </div>
                      <div className="flex items-center gap-1">
                        <Users className="h-3 w-3 text-muted-foreground" />
                        <span className="text-xs text-muted-foreground font-medium">
                          {s.TeamLabel || "未归属"}
                        </span>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm">{s.PoolID}</div>
                    <Badge variant="secondary" className="text-[10px] h-4 mt-1">
                      {s.SlicingMode}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center text-sm">
                      <Clock className="mr-1 h-3 w-3 text-muted-foreground" />
                      {formatDuration(s.StartTime, s.EndTime)}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <div className="flex items-center">
                        <span className={`text-sm font-bold ${s.GPUUtilAvg < 20 ? "text-rose-500" : "text-emerald-600"}`}>
                          {s.GPUUtilAvg.toFixed(1)}%
                        </span>
                        <span className="text-[10px] text-muted-foreground ml-1">(Avg)</span>
                      </div>
                      <div className="text-[10px] text-muted-foreground">
                        Peak: {s.GPUUtilMax.toFixed(1)}%
                      </div>
                      {s.GPUUtilAvg < 20 && s.EndTime && (
                        <div className="flex items-center text-[10px] text-rose-500 font-medium">
                          <AlertCircle className="h-2.5 w-2.5 mr-0.5" /> 低效运行
                        </div>
                      )}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm font-medium">{s.MemUsedMax} MiB</div>
                    <div className="w-16 bg-secondary h-1 rounded-full mt-1 overflow-hidden">
                      <div 
                        className="bg-blue-400 h-full" 
                        style={{ width: `${Math.min((s.MemUsedMax / 40960) * 100, 100)}%` }} 
                      />
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center text-sm">
                      <Zap className="mr-1 h-3 w-3 text-amber-500" />
                      {s.PowerUsageAvg.toFixed(1)} W
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="font-mono font-bold text-emerald-600">
                      ¥{s.CostAmount.toFixed(4)}
                    </div>
                  </TableCell>
                  <TableCell className="text-right">
                    <Badge variant={s.EndTime ? "outline" : "default"} className={s.EndTime ? "border-emerald-200 text-emerald-700 bg-emerald-50" : "bg-blue-500"}>
                      {s.EndTime ? "已结算" : "计费中"}
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

