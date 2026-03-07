import { NextResponse } from 'next/server';

// 这是一个 API 占位符，第一阶段主要验证预留
// 第四阶段完整 Dashboard 会通过 Redis / MySQL 拉取实时的 Pool 列表及用量

export async function GET() {
  // 模拟从 Redis 获取资源池信息
  const mockPools = [
    {
      pool_id: 'Train-H800-Full-Pool',
      nodes_count: 5,
      slicing_mode: 'Full',
      active_pods: 12,
      utilization: '85%',
    },
    {
      pool_id: 'Infer-A100-MIG-Pool',
      nodes_count: 2,
      slicing_mode: 'MIG',
      active_pods: 30,
      utilization: '72%',
    },
    {
      pool_id: 'Dev-V100-TS-Pool',
      nodes_count: 8,
      slicing_mode: 'TS',
      active_pods: 150,
      utilization: '40%',
    }
  ];

  return NextResponse.json({
    status: 'success',
    data: mockPools,
  });
}
