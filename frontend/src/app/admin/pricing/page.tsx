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
  Save, 
  RefreshCw, 
  Coins, 
  CircleDollarSign, 
  Scale, 
  Zap, 
  Cpu,
  Info
} from "lucide-react";

interface PoolPricing {
  PoolID: string;
  GPUModel: string;
  BasePricePerHour: number;
  SlicingWeightFull: number;
  SlicingWeightMIG: number;
  SlicingWeightMPS: number;
  SlicingWeightTS: number;
}

export default function PricingAdminPage() {
  const [pricingList, setPricingList] = useState<PoolPricing[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchPricing = () => {
    setLoading(true);
    fetch("http://localhost:8080/api/v2/pricing")
      .then((res) => res.json())
      .then((data) => {
        setPricingList(data || []);
        setLoading(false);
      });
  };

  useEffect(() => {
    fetchPricing();
  }, []);

  const handleUpdate = (poolID: string, data: PoolPricing) => {
    fetch(`http://localhost:8080/api/v2/pricing/${poolID}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    }).then(() => alert(`资源池 ${poolID} 的定价策略已更新并生效`));
  };

  const avgPrice = pricingList.length > 0 
    ? pricingList.reduce((acc, curr) => acc + curr.BasePricePerHour, 0) / pricingList.length 
    : 0;

  return (
    <div className="flex-1 space-y-6 p-8 pt-6 bg-slate-50/30 min-h-screen">
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h2 className="text-3xl font-bold tracking-tight text-slate-900 font-sans flex items-center gap-2">
            <CircleDollarSign className="h-8 w-8 text-amber-500" />
            池化定价管理
          </h2>
          <p className="text-sm text-muted-foreground flex items-center gap-1">
            <Info className="h-3 w-3" />
            配置各算力池的基准单价及切片模式下的计费权重系数
          </p>
        </div>
        <Button 
          onClick={fetchPricing} 
          variant="outline" 
          className="bg-white border-slate-200 shadow-sm"
        >
          <RefreshCw className={`mr-2 h-4 w-4 ${loading ? "animate-spin" : ""}`} /> 
          刷新定价表
        </Button>
      </div>

      {/* Pricing Logic Highlights */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card className="bg-gradient-to-br from-amber-500 to-orange-600 border-none shadow-md text-white">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium opacity-90">平均基准单价</CardTitle>
            <Coins className="h-4 w-4 opacity-80" />
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold">¥{avgPrice.toFixed(2)}</div>
            <p className="text-xs mt-1 opacity-70">按每小时/GPU 维度核算</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-blue-500">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600">计费颗粒度</CardTitle>
            <Scale className="h-4 w-4 text-blue-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-slate-800">秒级计费</div>
            <p className="text-xs text-muted-foreground mt-1">结算时自动根据权重进行折算</p>
          </CardContent>
        </Card>
        <Card className="bg-white border-slate-200 shadow-sm border-l-4 border-l-emerald-500">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-slate-600">虚拟化激励</CardTitle>
            <Zap className="h-4 w-4 text-emerald-500" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-slate-800">动态权重</div>
            <p className="text-xs text-muted-foreground mt-1">MIG/MPS 任务享有更低折扣</p>
          </CardContent>
        </Card>
      </div>

      <Card className="border-slate-200 shadow-sm overflow-hidden bg-white">
        <CardHeader className="border-b bg-slate-50/50">
          <CardTitle className="text-lg flex items-center text-slate-800">
            <Cpu className="mr-2 h-5 w-5 text-amber-600" />
            资源池单价与权重矩阵
          </CardTitle>
          <CardDescription className="text-sm text-muted-foreground">
            最终计费金额 = 运行时长(h) × 基准价 × 模式权重
          </CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader className="bg-slate-50/80">
              <TableRow>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase py-4 pl-6">资源池 ID / 硬件型号</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">基准单价 (¥/h)</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">MIG 权重</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">MPS 权重</TableHead>
                <TableHead className="text-xs font-semibold text-slate-600 uppercase">TS (超分) 权重</TableHead>
                <TableHead className="text-right text-xs font-semibold text-slate-600 uppercase pr-6">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pricingList.map((p, idx) => (
                <TableRow key={p.PoolID} className="hover:bg-slate-50 transition-colors border-b last:border-none">
                  <TableCell className="py-4 pl-6">
                    <div className="font-bold text-slate-800">{p.PoolID}</div>
                    <div className="text-xs text-slate-400 mt-1 flex items-center gap-1">
                      <Cpu className="h-3 w-3" />
                      {p.GPUModel}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <span className="text-slate-400 font-mono text-xs">¥</span>
                      <Input
                        type="number"
                        defaultValue={p.BasePricePerHour}
                        className="w-28 h-9 font-mono font-bold text-blue-600 border-slate-200 focus:ring-blue-500"
                        onChange={(e) => (pricingList[idx].BasePricePerHour = parseFloat(e.target.value))}
                      />
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <Input
                        type="number"
                        step="0.05"
                        defaultValue={p.SlicingWeightMIG}
                        className="w-20 h-8 text-xs border-slate-200"
                        onChange={(e) => (pricingList[idx].SlicingWeightMIG = parseFloat(e.target.value))}
                      />
                      <Badge variant="outline" className="text-[9px] px-1 h-4 scale-90 origin-left border-slate-200 text-slate-400">HARDWARE</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <Input
                        type="number"
                        step="0.05"
                        defaultValue={p.SlicingWeightMPS}
                        className="w-20 h-8 text-xs border-slate-200"
                        onChange={(e) => (pricingList[idx].SlicingWeightMPS = parseFloat(e.target.value))}
                      />
                      <Badge variant="outline" className="text-[9px] px-1 h-4 scale-90 origin-left border-slate-200 text-slate-400">SOFTWARE</Badge>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="space-y-1">
                      <Input
                        type="number"
                        step="0.05"
                        defaultValue={p.SlicingWeightTS}
                        className="w-20 h-8 text-xs border-slate-200"
                        onChange={(e) => (pricingList[idx].SlicingWeightTS = parseFloat(e.target.value))}
                      />
                      <Badge variant="outline" className="text-[9px] px-1 h-4 scale-90 origin-left border-emerald-100 text-emerald-500 bg-emerald-50/30">OVERSELL</Badge>
                    </div>
                  </TableCell>
                  <TableCell className="text-right pr-6">
                    <Button
                      size="sm"
                      onClick={() => handleUpdate(p.PoolID, pricingList[idx])}
                      className="bg-slate-900 hover:bg-slate-800 text-white shadow-sm h-9 px-4"
                    >
                      <Save className="mr-2 h-4 w-4" /> 更新策略
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Help Section */}
      <div className="rounded-lg bg-blue-50 border border-blue-100 p-4 flex items-start gap-3">
        <Info className="h-5 w-5 text-blue-500 mt-0.5" />
        <div className="text-sm text-muted-foreground">
          <p className="font-semibold text-sm mb-1">计费逻辑说明：</p>
          <ul className="list-disc list-inside space-y-1 text-xs text-muted-foreground">            <li><strong>Full 模式</strong>：默认权重为 1.0，即按整卡基准价计费。</li>
            <li><strong>权重系数</strong>：建议将共享模式（MIG/MPS/TS）的权重设为 0.1 ~ 0.8 之间，以鼓励用户使用切片资源。</li>
            <li><strong>即时生效</strong>：定价修改后，仅影响后续“指标缝合”引擎新处理的账单，已结算的历史账单不会回溯。</li>
          </ul>
        </div>
      </div>
    </div>
  );
}

