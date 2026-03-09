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
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Save, RefreshCw, Cpu, Tag } from "lucide-react";
import { ResourcePool } from "@/lib/types";

export default function ResourcePoolAdminPage() {
  const [pools, setPools] = useState<ResourcePool[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchPools = () => {
    setLoading(true);
    fetch("http://localhost:8080/api/v2/pools")
      .then((res) => res.json())
      .then((data) => {
        setPools(data || []);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchPools();
  }, []);

  const handleUpdate = (id: string, index: number) => {
    fetch(`http://localhost:8080/api/v2/pools/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(pools[index]),
    }).then(() => alert("资源池业务信息已更新"));
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">资源池资产管理</h2>
        <Button onClick={fetchPools} variant="outline">
          <RefreshCw className="mr-2 h-4 w-4" /> 刷新发现
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg flex items-center">
            <Cpu className="mr-2 h-5 w-5 text-blue-500" />
            已感知的逻辑资产清单
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pool ID / 业务别名</TableHead>
                <TableHead>场景 & 优先级</TableHead>
                <TableHead>硬件型号 & 特性</TableHead>
                <TableHead>虚拟化 / 定价逻辑</TableHead>
                <TableHead>资产描述</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((p, idx) => (
                <TableRow key={p.ID}>
                  <TableCell className="max-w-[200px]">
                    <div className="text-[10px] font-mono text-muted-foreground mb-1">{p.ID}</div>
                    <Input
                      defaultValue={p.Name}
                      className="h-8 font-medium"
                      onChange={(e) => (pools[idx].Name = e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="w-[150px] space-y-2">
                    <Select 
                      defaultValue={p.Scene}
                      onValueChange={(val) => (pools[idx].Scene = val)}
                    >
                      <SelectTrigger className="h-8 text-xs">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="大模型预训练">预训练</SelectItem>
                        <SelectItem value="模型微调">微调</SelectItem>
                        <SelectItem value="核心推理">核心推理</SelectItem>
                        <SelectItem value="小模型推理">小模型推理</SelectItem>
                        <SelectItem value="研发调试">研发调试</SelectItem>
                      </SelectContent>
                    </Select>
                    <div className="flex gap-1">
                      <Badge 
                        variant={p.Priority === "High" ? "default" : "outline"}
                        className="text-[10px] cursor-pointer"
                        onClick={() => {
                          const next = p.Priority === "High" ? "Low" : "High";
                          pools[idx].Priority = next;
                          setPools([...pools]);
                        }}
                      >
                        {p.Priority}
                      </Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm font-medium">{p.GPUModel}</div>
                    <div className="flex flex-wrap gap-1 mt-1">
                      {p.HardwareFeatures.split(",").map(f => (
                        <Badge key={f} variant="secondary" className="text-[9px] px-1 h-4">{f}</Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Tag className="h-3 w-3 text-muted-foreground" />
                      <span className="text-xs">{p.SlicingMode}</span>
                    </div>
                    <Input
                      defaultValue={p.PricingLogic}
                      className="h-7 text-[10px] mt-1"
                      onChange={(e) => (pools[idx].PricingLogic = e.target.value)}
                    />
                  </TableCell>
                  <TableCell>
                    <textarea
                      defaultValue={p.Description}
                      className="w-full text-xs p-1 border rounded min-h-[60px]"
                      onChange={(e) => (pools[idx].Description = e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      onClick={() => handleUpdate(p.ID, idx)}
                    >
                      <Save className="mr-2 h-4 w-4" /> 保存
                    </Button>
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
