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
import { Save, RefreshCw } from "lucide-react";

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
    }).then(() => alert("定价已更新"));
  };

  return (
    <div className="flex-1 space-y-4 p-8 pt-6">
      <div className="flex items-center justify-between">
        <h2 className="text-3xl font-bold tracking-tight">资源池定价管理</h2>
        <Button onClick={fetchPricing} variant="outline">
          <RefreshCw className="mr-2 h-4 w-4" /> 刷新数据
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>池化资源基准价 (每小时/GPU)</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>资源池 ID</TableHead>
                <TableHead>显卡型号</TableHead>
                <TableHead>基准价 (¥/h)</TableHead>
                <TableHead>MIG 权重</TableHead>
                <TableHead>MPS 权重</TableHead>
                <TableHead>TS 权重</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pricingList.map((p, idx) => (
                <TableRow key={p.PoolID}>
                  <TableCell className="font-medium">{p.PoolID}</TableCell>
                  <TableCell>{p.GPUModel}</TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      defaultValue={p.BasePricePerHour}
                      className="w-24"
                      onChange={(e) => (pricingList[idx].BasePricePerHour = parseFloat(e.target.value))}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      defaultValue={p.SlicingWeightMIG}
                      className="w-20"
                      onChange={(e) => (pricingList[idx].SlicingWeightMIG = parseFloat(e.target.value))}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      defaultValue={p.SlicingWeightMPS}
                      className="w-20"
                      onChange={(e) => (pricingList[idx].SlicingWeightMPS = parseFloat(e.target.value))}
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      type="number"
                      defaultValue={p.SlicingWeightTS}
                      className="w-20"
                      onChange={(e) => (pricingList[idx].SlicingWeightTS = parseFloat(e.target.value))}
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      size="sm"
                      onClick={() => handleUpdate(p.PoolID, pricingList[idx])}
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
