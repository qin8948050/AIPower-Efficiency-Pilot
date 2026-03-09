"use client";

import { usePathname } from "next/navigation";
import { AppSidebar } from "@/components/app-sidebar";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { Separator } from "@/components/ui/separator";

const BREADCRUMBS: Record<string, { main: string; sub: string }> = {
  "/": { main: "核心面板", sub: "资源池实时大屏" },
  "/billing": { main: "成本分摊中心", sub: "概览看板" },
  "/billing/sessions": { main: "成本分摊中心", sub: "Pod 级账单明细" },
  "/billing/teams": { main: "成本分摊中心", sub: "业务维度分摊" },
  "/billing/pools": { main: "成本分摊中心", sub: "资源池效能量化" },
  "/admin/pricing": { main: "配置中心", sub: "池化定价管理" },
};

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const breadcrumb = BREADCRUMBS[pathname] || { main: "核心面板", sub: "实时监控" };

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-16 shrink-0 items-center gap-2 transition-[width,height] ease-linear group-has-[[data-collapsible=icon]]/sidebar-wrapper:h-12 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator orientation="vertical" className="mr-2 h-4" />
          <div className="flex items-center gap-2">
            <span className="text-sm font-semibold">{breadcrumb.main}</span>
            <span className="text-xs text-muted-foreground">/ {breadcrumb.sub}</span>
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 pt-4">
          {children}
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}

