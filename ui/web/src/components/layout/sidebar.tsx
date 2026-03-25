"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  ShieldAlert,
  Bell,
  GitBranch,
  Shield,
  Radio,
  Activity,
} from "lucide-react";

const navItems = [
  { href: "/", icon: LayoutDashboard, label: "Dashboard" },
  { href: "/incidents", icon: ShieldAlert, label: "Incidents" },
  { href: "/alerts", icon: Bell, label: "Alerts" },
  { href: "/graph", icon: GitBranch, label: "Attack Graph" },
  { href: "/compliance", icon: Shield, label: "Compliance" },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="fixed inset-y-0 left-0 z-40 w-56 flex flex-col bg-viola-surface border-r border-viola-border">
      {/* Brand */}
      <div className="flex items-center gap-2.5 px-4 py-4 border-b border-viola-border">
        <div className="flex items-center justify-center w-7 h-7 rounded-md bg-viola-accent/10 border border-viola-accent/30">
          <Radio className="h-4 w-4 text-viola-accent" />
        </div>
        <div>
          <p className="text-sm font-bold text-viola-text tracking-wide">VIOLA</p>
          <p className="text-[10px] text-viola-muted uppercase tracking-widest">XDR Console</p>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 px-2 py-4 space-y-0.5">
        {navItems.map(({ href, icon: Icon, label }) => {
          const isActive =
            href === "/"
              ? pathname === "/"
              : pathname.startsWith(href);

          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors",
                isActive
                  ? "bg-viola-accent/10 text-viola-accent border border-viola-accent/20"
                  : "text-viola-muted hover:text-viola-text hover:bg-viola-border/50"
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </Link>
          );
        })}
      </nav>

      {/* Status indicator */}
      <div className="px-4 py-3 border-t border-viola-border">
        <div className="flex items-center gap-2">
          <Activity className="h-3.5 w-3.5 text-viola-muted" />
          <span className="text-[11px] text-viola-muted">Pipeline</span>
          <span className="ml-auto flex items-center gap-1 text-[10px] text-emerald-400">
            <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse" />
            Live
          </span>
        </div>
        <div className="mt-2 flex items-center gap-2">
          <div className="h-1 flex-1 rounded-md bg-viola-border overflow-hidden">
            <div className="h-full w-[72%] bg-viola-accent/60 rounded-md" />
          </div>
          <span className="text-[10px] text-viola-muted font-mono">72%</span>
        </div>
      </div>
    </aside>
  );
}
