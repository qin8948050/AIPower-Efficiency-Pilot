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
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from "recharts";
import { DailyBillingSnapshot } from "@/lib/types";

export default function TeamsPage() {
  const [snapshots, setSnapshots] = useState<DailyBillingSnapshot[]>([]);

  useEffect(() => {
    fetch("http://localhost:8080/api/v2/billing/daily")
      .then((res) => res.json())
      .then((data) => setSnapshots(data || []));
  }, []);

  // 按 TeamLabel 聚合
  const teamMap = new Map<string, number>();
  snapshots.forEach((s) => {
    const team = s.TeamLabel || "未打标业务";
    teamMap.set(team, (teamMap.get(team) || 0) + s.TotalCost);
  });

  const chartData = Array.from(teamMap.entries())
    .map(([name, cost]) => ({ name, cost }))
    .sort((a, b) => b.cost - a.cost);

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">业务维度分摊</h2>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>团队成本概览</CardTitle>
            <CardDescription>按业务部门 (Team Label) 进行成本核算</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="h-[400px]">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={chartData} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" horizontal={false} />
                  <XAxis type="number" hide />
                  <YAxis
                    dataKey="name"
                    type="category"
                    fontSize={12}
                    width={100}
                    tickLine={false}
                    axisLine={false}
                  />
                  <Tooltip
                    formatter={(value: number) => `¥${value.toFixed(2)}`}
                    contentStyle={{ borderRadius: "8px" }}
                  />
                  <Bar dataKey="cost" fill="#10b981" radius={[0, 4, 4, 0]} barSize={20} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </CardContent>
        </Card>

        <Card className="col-span-1">
          <CardHeader>
            <CardTitle>明细说明</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {chartData.map((t) => (
                <div key={t.name} className="flex items-center">
                  <div className="flex-1">
                    <div className="font-medium">{t.name}</div>
                    <div className="text-xs text-muted-foreground">占比 {((t.cost / (snapshots.reduce((a, b) => a + b.TotalCost, 0) || 1)) * 100).toFixed(1)}%</div>
                  </div>
                  <div className="font-mono">¥{t.cost.toFixed(2)}</div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
