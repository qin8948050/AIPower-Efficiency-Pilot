"use client";

import * as React from "react";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Wallet,
  BrainCircuit,
  Settings,
  ShieldCheck,
  ChevronRight,
  TrendingDown,
  Layout,
  Terminal,
  Box,
} from "lucide-react";

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarRail,
} from "@/components/ui/sidebar";

const data = {
  navMain: [
    {
      title: "核心面板",
      url: "#",
      icon: LayoutDashboard,
      isActive: true,
      items: [
        {
          title: "实时资源池大屏",
          url: "/",
          icon: Layout,
        },
      ],
    },
    {
      title: "成本分摊中心",
      url: "#",
      icon: Wallet,
      items: [
        {
          title: "概览看板",
          url: "/billing",
          icon: Wallet,
        },
        {
          title: "Pod 级账单明细",
          url: "/billing/sessions",
          icon: Layout,
        },
        {
          title: "业务维度分摊",
          url: "/billing/teams",
          icon: Wallet,
        },
        {
          title: "资源池效能量化",
          url: "/billing/pools",
          icon: Box,
        },
      ],
    },
    {
      title: "智能治理",
      url: "#",
      icon: BrainCircuit,
      items: [
        {
          title: "AI 诊断报告",
          url: "/ai-report",
          icon: ShieldCheck,
        },
        {
          title: "治理执行中心",
          url: "/pilot",
          icon: TrendingDown,
        },
      ],
    },
    {
      title: "配置中心",
      url: "#",
      icon: Settings,
      items: [
        {
          title: "池化定价管理",
          url: "/admin/pricing",
          icon: Wallet,
        },
        {
          title: "系统参数设定",
          url: "/settings/system",
          icon: Terminal,
        },
      ],
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const pathname = usePathname();

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <div className="flex items-center gap-2 px-4 py-2">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <LayoutDashboard className="h-5 w-5" />
          </div>
          <div className="flex flex-col gap-0.5 overflow-hidden">
            <span className="font-semibold leading-none truncate">
              FinOps Pilot
            </span>
            <span className="text-xs text-muted-foreground truncate">
              AIPower-Efficiency
            </span>
          </div>
        </div>
      </SidebarHeader>
      <SidebarContent>
        {data.navMain.map((group) => (
          <SidebarGroup key={group.title}>
            <SidebarGroupLabel>{group.title}</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {group.items.map((item) => {
                  const isActive = pathname === item.url;
                  return (
                    <SidebarMenuItem key={item.title}>
                      <SidebarMenuButton
                        asChild
                        tooltip={item.title}
                        isActive={isActive}
                        className={isActive ? "bg-sidebar-primary text-sidebar-primary-foreground font-bold shadow-sm hover:bg-sidebar-primary hover:text-sidebar-primary-foreground" : ""}
                      >
                        <a href={item.url}>
                          {item.icon && <item.icon className={isActive ? "text-sidebar-primary-foreground" : ""} />}
                          <span>{item.title}</span>
                        </a>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}
