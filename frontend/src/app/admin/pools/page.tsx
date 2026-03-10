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
import { 
  Save, 
  RefreshCw, 
  Cpu, 
  Tag, 
  Layers, 
  Layout, 
  ShieldAlert, 
  HardDrive,
  Info,
  Box
} from "lucide-react";
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

  const highPriorityCount = pools.filter(p => p.Priority === "High").length;
  const totalPools = pools.length;

  return (
    <div className="flex-1 space-y-6 p-8 pt-6 bg-slate-50/30 min-h-screen">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 flex items-center gap-2">
            <Box className="h-8 w-8 text-blue-600" />
            资源池资产管理
          </h2>
          <p className="text-sm text-muted-foreground flex items-center gap-1">
            <Info className="h-3 w-3" />
            基于 K8s 标签自动感知逻辑资产，支持补充财务与业务维度的元数据
          </p>
        </div>
        <Button 
          onClick={fetchPools} 
          variant="outline" 
          className="bg-white hover:bg-slate-50 border-slate-200 shadow-sm"
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} /> 
          同步发现结果
        </Button>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <Card className="bg-gradient-to-br from-blue-500 to-blue-600 border-none shadow-md text-white">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium opacity-90">感知的总资产</CardTitle>
            <Layers className="h-4 w-4 opacity-80" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{totalPools}</div>
            <p className="text-xs mt-1 opacity-70">全集群逻辑算力池</p>
          </CardContent>
        </Card>
        <Card className="bg-gradient-to-br from-emerald-500 to-emerald-600 border-none shadow-md text-white">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium opacity-90">生产级池 (High)</CardTitle>
            <ShieldAlert className="h-4 w-4 opacity-80" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">{highPriorityCount}</div>
            <p className="text-xs mt-1 opacity-70">高优先级治理目标</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600">最新发现硬件</CardTitle>
            <Cpu className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-lg font-bold truncate text-slate-800">
              {pools.length > 0 ? pools[0].GPUModel : "--"}
            </div>
            <p className="text-xs text-muted-foreground mt-1">自动识别自 K8s Labels</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600">资产入库状态</CardTitle>
            <Layout className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-emerald-600">一致</div>
            <p className="text-xs text-muted-foreground mt-1">与底层 K8s 实时同步</p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-slate-200 shadow-sm overflow-hidden bg-white">
        <CardHeader className="border-b bg-slate-50/50">
          <CardTitle className="text-lg flex items-center text-slate-800">
            <HardDrive className="mr-2 h-5 w-5 text-blue-600" />
            资产清单与业务映射
          </CardTitle>
          <CardDescription className="text-sm text-muted-foreground">
            将物理算力标识映射为业务部门、定价策略及治理优先级
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader className="bg-slate-50/80">
              <TableRow>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase py-4 pl-6">资源池标识 / 别名</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">应用场景 & 权重</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">硬件架构特性</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">计费逻辑</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">资产备注</TableHead>
                <TableHead className="text-right text-xs font-semibold text-slate-600 uppercase pr-6">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pools.map((p, idx) => (
                <TableRow key={p.ID} className="hover:bg-slate-50 transition-colors border-b last:border-none">
                  <TableCell className="max-w-[240px] py-4 pl-6">
                    <div className="text-[10px] font-mono text-slate-400 mb-1 bg-slate-100 px-1.5 py-0.5 rounded w-fit">{p.ID}</div>
                    <Input
                      defaultValue={p.Name}
                      className="h-9 font-semibold text-slate-800 border-slate-200 focus:ring-blue-500"
                      onChange={(e) => (pools[idx].Name = e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="w-[180px] space-y-2">
                    <Select 
                      defaultValue={p.Scene}
                      onValueChange={(val) => (pools[idx].Scene = val)}
                    >
                      <SelectTrigger className="h-9 text-xs border-slate-200">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="大模型预训练">大模型预训练</SelectItem>
                        <SelectItem value="模型微调">模型微调</SelectItem>
                        <SelectItem value="核心推理">核心推理</SelectItem>
                        <SelectItem value="小模型推理">小模型推理</SelectItem>
                        <SelectItem value="研发调试">研发调试</SelectItem>
                      </SelectContent>
                    </Select>
                    <div className="flex gap-2">
                      <Badge 
                        variant={p.Priority === "High" ? "default" : "outline"}
                        className={`text-[10px] cursor-pointer px-2 py-0.5 ${p.Priority === "High" ? "bg-red-500 hover:bg-red-600" : "text-slate-500 border-slate-300 hover:bg-slate-100"}`}
                        onClick={() => {
                          const next = p.Priority === "High" ? "Low" : "High";
                          pools[idx].Priority = next;
                          setPools([...pools]);
                        }}
                      >
                        {p.Priority} Priority
                      </Badge>
                      <Badge variant="secondary" className="text-[10px] bg-blue-50 text-blue-700 border-blue-100">{p.SlicingMode}</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="text-sm font-bold text-slate-700 flex items-center gap-1.5">
                      <Cpu className="h-3 w-3 text-slate-400" />
                      {p.GPUModel}
                    </div>
                    <div className="flex flex-wrap gap-1.5 mt-2">
                      {p.HardwareFeatures ? p.HardwareFeatures.split(",").map(f => (
                        <span key={f} className="text-[9px] px-1.5 py-0.5 bg-amber-50 text-amber-700 border border-amber-100 rounded-sm font-medium">{f}</span>
                      )) : <span className="text-[9px] text-slate-400 italic">无特殊特性</span>}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1.5 mb-2">
                      <Tag className="h-3.5 w-3.5 text-blue-500" />
                      <span className="text-xs font-medium text-slate-600">定价逻辑</span>
                    </div>
                    <Input
                      defaultValue={p.PricingLogic}
                      className="h-8 text-[11px] border-slate-200 focus:ring-blue-500"
                      onChange={(e) => (pools[idx].PricingLogic = e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="max-w-[200px]">
                    <textarea
                      defaultValue={p.Description}
                      placeholder="点击编辑备注..."
                      className="w-full text-xs p-2 border border-slate-200 rounded-md min-h-[70px] bg-slate-50/30 focus:bg-white focus:ring-1 focus:ring-blue-500 outline-none transition-all"
                      onChange={(e) => (pools[idx].Description = e.target.value)}
                    />
                  </TableCell>
                  <TableCell className="text-right pr-6">
                    <Button
                      size="sm"
                      onClick={() => handleUpdate(p.ID, idx)}
                      className="bg-blue-600 hover:bg-blue-700 text-white shadow-sm h-9 px-4"
                    >
                      <Save className="mr-2 h-4 w-4" /> 永久保存
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

