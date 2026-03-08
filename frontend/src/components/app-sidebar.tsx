"use client";

import * as React from "react";
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
        {
          title: "成本分摊中心",
          url: "/billing",
          icon: Wallet,
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
          url: "/settings/pricing",
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
                {group.items.map((item) => (
                  <SidebarMenuItem key={item.title}>
                    <SidebarMenuButton asChild tooltip={item.title}>
                      <a href={item.url}>
                        {item.icon && <item.icon />}
                        <span>{item.title}</span>
                      </a>
                    </SidebarMenuButton>
                  </SidebarMenuItem>
                ))}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>
      <SidebarRail />
    </Sidebar>
  );
}
