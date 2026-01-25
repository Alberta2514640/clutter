"use client";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { BarChart3, BookTemplate, ChevronLeft, ChevronRight, FolderOpen, HelpCircle, LayoutDashboard, Plus, Settings, Sparkles } from "lucide-react";
import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

interface SidebarProps {
  className?: string;
}

type StoredGoogleDataShapeA = {
  user_data: {
    full_name: string;
    email: string;
    picture_url: string;
    uuid: string;
    created_at: string;
  };
};

type StoredGoogleDataShapeB = {
  data: {
    full_name: string;
    email: string;
    picture_url: string;
    uuid: string;
    created_at: string;
  };
};

type UserProfile = {
  full_name: string;
  email: string;
  picture_url: string;
  uuid: string;
  created_at: string;
};

function pickProfile(obj: unknown): UserProfile | null {
  if (!obj || typeof obj !== "object") return null;

  const o = obj as Partial<StoredGoogleDataShapeA & StoredGoogleDataShapeB>;

  const rawProfile = o.user_data ?? o.data;
  if (!rawProfile) return null;

  const { full_name, email, picture_url, uuid, created_at } = rawProfile;

  if (typeof full_name !== "string" || typeof email !== "string") return null;

  return {
    full_name,
    email,
    picture_url: typeof picture_url === "string" ? picture_url : "",
    uuid: typeof uuid === "string" ? uuid : "",
    created_at: typeof created_at === "string" ? created_at : "",
  };
}

export default function DashboardSidebar({ className }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(true);
  const pathname = usePathname();

  const [profile, setProfile] = useState<UserProfile | null>(null);

  useEffect(() => {
    const raw = localStorage.getItem("google_data");
    if (!raw) return;

    try {
      const parsed = JSON.parse(raw) as unknown;
      const p = pickProfile(parsed);
      if (p) setProfile(p);
    } catch (e) {
      console.error("Failed to parse google_data:", e);
    }
  }, []);

  const initials = useMemo(() => {
    const name = profile?.full_name?.trim() || "";
    if (!name) return "U";
    const parts = name.split(/\s+/).filter(Boolean);
    const first = parts[0]?.[0] ?? "U";
    const last = parts.length > 1 ? parts[parts.length - 1]?.[0] : "";
    return (first + last).toUpperCase();
  }, [profile]);

  const navItems = [{ icon: LayoutDashboard, label: "Overview", href: "/dashboard" }];

  const projects = [
    { id: "1", name: "Web Application", icon: "🌐" },
    { id: "2", name: "Data Pipeline", icon: "📊" },
    { id: "3", name: "Monitoring Stack", icon: "📈" },
  ];

  const bottomItems = [
    { icon: Settings, label: "Admin Panel", href: "/settings/organization/manage" },
    { icon: BookTemplate, label: "Templates", href: "/templates" },
    { icon: BarChart3, label: "Insights", href: "/insights" },
    { icon: HelpCircle, label: "Help", href: "/help" },
    { icon: Sparkles, label: "What's New", href: "/whats-new" },
  ];

  const handleExpand = () => {
    if (collapsed) setCollapsed(false);
  };

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    setCollapsed((v) => !v);
  };

  return (
    <aside onClick={handleExpand} className={cn("relative flex flex-col border-r border-slate-800/50 bg-slate-900/40 backdrop-blur-xl transition-all duration-300", collapsed ? "w-16 cursor-pointer" : "w-64", className)}>
      {/* Expand/Collapse Button */}
      <button onClick={handleToggle} className="absolute -right-3 top-1/2 z-10 p-1.5 rounded-full bg-slate-800 border border-slate-700 hover:bg-slate-700 transition-colors shadow-lg">
        {collapsed ? <ChevronRight className="w-4 h-4 text-gray-400" /> : <ChevronLeft className="w-4 h-4 text-gray-400" />}
      </button>

      {/* Logo Section */}
      <div className="flex items-center justify-center p-4 border-b border-slate-800/50">
        {collapsed ? <Image src="/logos/logo.png" alt="Clutter" width={80} height={80} className="object-contain" /> : <Image src="/logos/logo_text.png" alt="Clutter" width={80} height={80} className="object-contain" />}
      </div>

      {/* Main Navigation */}
      <nav className="flex-1 overflow-y-auto p-3 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={(e) => e.stopPropagation()}
              className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", isActive ? "bg-teal-500/10 text-teal-400" : "text-gray-400 hover:bg-slate-800/50 hover:text-white", collapsed && "justify-center")}>
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && <span className="text-sm font-medium">{item.label}</span>}
            </Link>
          );
        })}

        {/* Projects Section */}
        {!collapsed && (
          <div className="pt-6" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-3 pb-2">
              <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Projects</span>
              <Button size="sm" variant="ghost" className="h-6 w-6 p-0 hover:bg-slate-800/50" asChild>
                <Link href="/projects/new">
                  <Plus className="w-4 h-4 text-gray-400" />
                </Link>
              </Button>
            </div>

            <div className="space-y-1">
              {projects.map((project) => (
                <Link
                  key={project.id}
                  href={`/projects/${project.id}/diagrams`}
                  className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", pathname.includes(`/projects/${project.id}`) ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/30 hover:text-white")}>
                  <span className="text-lg flex-shrink-0">{project.icon}</span>
                  <span className="text-sm truncate">{project.name}</span>
                </Link>
              ))}
            </div>
          </div>
        )}

        {/* Collapsed state icons for projects */}
        {collapsed && (
          <div className="pt-6 space-y-1">
            <div className="flex items-center justify-center px-2 pb-2">
              <FolderOpen className="text-gray-500 px-0.5" />
            </div>
            {projects.slice(0, 3).map((project) => (
              <Link
                key={project.id}
                href={`/projects/${project.id}`}
                onClick={(e) => e.stopPropagation()}
                className={cn("flex items-center justify-center px-3 py-2 rounded-lg transition-all", pathname.includes(`/projects/${project.id}`) ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/30 hover:text-white")}>
                <span className="text-lg">{project.icon}</span>
              </Link>
            ))}
          </div>
        )}
      </nav>

      {/* Bottom Navigation */}
      <div className="border-t border-slate-800/50 p-3 space-y-1">
        {bottomItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              onClick={(e) => e.stopPropagation()}
              className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", isActive ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/50 hover:text-white", collapsed && "justify-center")}>
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && <span className="text-sm">{item.label}</span>}
            </Link>
          );
        })}

        {/* User Profile */}
        <div onClick={(e) => e.stopPropagation()} className={cn("flex items-center gap-3 px-3 py-2 mt-2 rounded-lg hover:bg-slate-800/50 cursor-pointer transition-all", collapsed && "justify-center")}>
          <div className="w-8 h-8 rounded-full overflow-hidden bg-slate-800 flex items-center justify-center flex-shrink-0">
            {profile?.picture_url ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={profile.picture_url} alt="Profile" className="w-full h-full object-cover" />
            ) : (
              <span className="text-white font-semibold text-sm">{initials}</span>
            )}
          </div>

          {!collapsed && (
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-white truncate">{profile?.full_name ?? "Loading..."}</p>
              <p className="text-xs text-gray-500 truncate">{profile?.email ?? ""}</p>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}
