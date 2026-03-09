export interface DailyBillingSnapshot {
  ID: number;
  SnapshotDate: string;
  PoolID: string;
  Namespace: string;
  TeamLabel: string;
  TotalCost: number;
  AvgUtilP95: number;
  MaxMemGiB: number;
  PodSessionCount: number;
}

export interface LifeTrace {
  ID: number;
  PodUID: string;
  Namespace: string;
  PodName: string;
  NodeName: string;
  PoolID: string;
  SlicingMode: string;
  StartTime: string;
  EndTime: string | null;
  TeamLabel: string;
  ProjectLabel: string;
  GPUUtilAvg: number;
  GPUUtilMax: number;
  MemUsedMax: number;
  PowerUsageAvg: number;
  CostAmount: number;
}

export interface ResourcePool {
  ID: string;
  Name: string;
  Scene: string;
  GPUModel: string;
  HardwareFeatures: string;
  SlicingMode: string;
  PricingLogic: string;
  Priority: string;
  Description: string;
  UpdatedAt: string;
}
