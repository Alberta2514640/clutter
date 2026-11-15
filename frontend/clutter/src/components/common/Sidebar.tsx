"use client";

import ClutterLogo from "@/components/common/ClutterLogo";
import { Button } from "@/components/ui/button";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { useProjects } from "@/lib/features/projects/hooks";
import { useLogout, useMe } from "@/lib/features/user/hooks";
import { cn } from "@/lib/utils";
import { Building2, Cable, ChevronLeft, ChevronRight, FolderOpen, LayoutDashboard, LogOut, Menu, Plus, Settings, X } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useState } from "react";

interface SidebarProps {
  className?: string;
}

// A page inside one project belongs to that project's organization, so it
// cannot stay open when the user switches to another organization.
const isProjectPage = (pathname: string) => pathname.startsWith("/projects/") && pathname !== "/projects/new";

// Below md the sidebar is a drawer, closed by default, opened from the top
// bar's menu button. Its parent lays out as a column below md and a row above.
export default function DashboardSidebar({ className }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(false);
  const pathname = usePathname();
  // The drawer stays open only on the page it was opened on, so any
  // navigation closes it.
  const [drawerPath, setDrawerPath] = useState<string | null>(null);
  const drawerOpen = drawerPath === pathname;
  const closeDrawer = () => setDrawerPath(null);
  const openDrawer = () => {
    setCollapsed(false);
    setDrawerPath(pathname);
  };

  useEffect(() => {
    if (!drawerOpen) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setDrawerPath(null);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [drawerOpen]);
  const router = useRouter();
  const logout = useLogout();

  const meQ = useMe();
  const user = meQ.data ?? null;
  const token = user?.token ?? null;
  const currentOrg = useCurrentOrganization(token);
  const orgId = currentOrg.organizationId;

  const projectsQ = useProjects(token, orgId);
  const projects = projectsQ.data ?? [];

  const navItems = [{ icon: LayoutDashboard, label: "Overview", href: "/dashboard" }];

  const bottomItems = [
    { icon: Cable, label: "Connections", href: "/settings/organization/connections" },
    { icon: Settings, label: "Organization settings", href: "/settings/organization/manage" },
  ];

  const getUserInitials = (displayName?: string, email?: string) => {
    if (displayName) {
      const names = displayName.trim().split(/\s+/);
      if (names.length >= 2) return `${names[0][0]}${names[1][0]}`.toUpperCase();
      return displayName.slice(0, 2).toUpperCase();
    }
    if (email) return email.slice(0, 2).toUpperCase();
    return "??";
  };

  const handleSelectOrganization = (organizationId: string) => {
    currentOrg.select(organizationId);
    if (isProjectPage(pathname)) router.push("/dashboard");
  };

  const handleLogout = () => {
    logout();
    router.push("/login");
    router.refresh();
  };

  return (
    <>
      {/* Top bar below md: the menu button and the logo */}
      <div className="flex h-14 shrink-0 items-center gap-3 border-b border-slate-800/50 bg-slate-950/80 px-4 backdrop-blur-xl md:hidden">
        <button
          type="button"
          onClick={openDrawer}
          aria-label="Open menu"
          aria-expanded={drawerOpen}
          aria-controls="app-sidebar"
          className="rounded-md p-1.5 text-slate-300 hover:bg-slate-800 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
          <Menu className="h-5 w-5" aria-hidden="true" />
        </button>
        <Link href="/dashboard" aria-label="Clutter dashboard">
          <ClutterLogo size={26} withName />
        </Link>
      </div>

      {drawerOpen && <div aria-hidden="true" className="fixed inset-0 z-40 bg-black/60 backdrop-blur-sm md:hidden" onClick={closeDrawer} />}

    <aside
      id="app-sidebar"
      className={cn(
        // Closed below md, the drawer is also invisible, so Tab and screen
        // readers skip it; visibility flips after the slide, not before.
        "fixed inset-y-0 left-0 z-50 flex w-64 shrink-0 flex-col border-r border-slate-800/50 bg-slate-950 transition-[translate,transform,visibility] duration-300 md:relative md:z-auto md:h-screen md:translate-x-0 md:bg-slate-900/40 md:backdrop-blur-xl md:transition-all",
        drawerOpen ? "translate-x-0" : "-translate-x-full max-md:invisible",
        collapsed && "md:w-16",
        className,
      )}>
      {/* Expand / Collapse toggle (md and up) */}
      <button
        type="button"
        onClick={() => setCollapsed((v) => !v)}
        aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"}
        aria-expanded={!collapsed}
        className="absolute -right-3 top-1/2 z-10 hidden p-1.5 rounded-full bg-slate-800 border border-slate-700 hover:bg-slate-700 transition-colors shadow-lg md:block">
        {collapsed ? <ChevronRight className="w-4 h-4 text-gray-400" aria-hidden="true" /> : <ChevronLeft className="w-4 h-4 text-gray-400" aria-hidden="true" />}
      </button>

      {/* Logo, and the drawer's close button below md */}
      <div className="flex items-center justify-between p-4 border-b border-slate-800/50 md:justify-center">
        {collapsed ? <ClutterLogo size={32} /> : <ClutterLogo size={32} withName />}
        <button
          type="button"
          onClick={closeDrawer}
          aria-label="Close menu"
          className="rounded-md p-1.5 text-slate-400 hover:bg-slate-800 hover:text-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300 md:hidden">
          <X className="h-5 w-5" aria-hidden="true" />
        </button>
      </div>

      {/* Current organization */}
      {!collapsed && (
        <div className="border-b border-slate-800/50 px-3 py-3">
          <label htmlFor="current-organization" className="mb-1 flex items-center gap-2 px-1 text-xs font-semibold uppercase tracking-wider text-gray-500">
            <Building2 className="h-3.5 w-3.5" aria-hidden="true" />
            Organization
          </label>
          {currentOrg.isLoading ? (
            <div className="px-1 py-1.5 text-xs text-gray-500">Loading organizations…</div>
          ) : currentOrg.isError ? (
            <div className="px-1 py-1.5 text-xs text-red-400">Could not load your organizations.</div>
          ) : currentOrg.organizations.length === 0 ? (
            <div className="px-1 py-1.5 text-xs text-gray-500">No organization yet</div>
          ) : (
            <select
              id="current-organization"
              value={orgId ?? ""}
              onChange={(e) => handleSelectOrganization(e.target.value)}
              className="w-full rounded-lg border border-slate-700 bg-slate-900/60 px-2 py-1.5 text-sm text-white outline-none focus:border-teal-500/60">
              {currentOrg.organizations.map((o) => (
                <option key={o.id} value={o.id}>
                  {o.name}
                </option>
              ))}
            </select>
          )}
          <Link href="/settings/organization/create-org" className="mt-2 flex items-center gap-1.5 px-1 text-xs text-gray-400 hover:text-white">
            <Plus className="h-3.5 w-3.5" aria-hidden="true" />
            New organization
          </Link>
        </div>
      )}

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto p-3 space-y-1">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              aria-label={collapsed ? item.label : undefined}
              className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", isActive ? "bg-teal-500/10 text-teal-400" : "text-gray-400 hover:bg-slate-800/50 hover:text-white", collapsed && "justify-center")}>
              <Icon className="w-5 h-5 flex-shrink-0" aria-hidden="true" />
              {!collapsed && <span className="text-sm font-medium">{item.label}</span>}
            </Link>
          );
        })}

        {/* Projects (expanded) */}
        {!collapsed && (
          <div className="pt-6">
            <div className="flex items-center justify-between px-3 pb-2">
              <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">Projects</span>

              {/* The dashboard has its own New project button; one is enough there. */}
              {pathname !== "/dashboard" && (
                <Button size="sm" variant="ghost" className="h-6 w-6 p-0 hover:bg-slate-800/50" asChild>
                  <Link href="/projects/new" aria-label="New project">
                    <Plus className="w-4 h-4 text-gray-400" aria-hidden="true" />
                  </Link>
                </Button>
              )}
            </div>

            <div className="space-y-1">
              {projectsQ.isLoading && projects.length === 0 ? (
                <div className="px-3 py-2 text-xs text-gray-500 text-center">Loading projects…</div>
              ) : projectsQ.isError ? (
                <div className="px-3 py-2 text-xs text-red-400 text-center">Could not load projects.</div>
              ) : projects.length === 0 ? (
                <div className="px-3 py-2 text-xs text-gray-500 text-center">No projects yet</div>
              ) : (
                projects.map((project) => (
                  <Link
                    key={project.id}
                    href={`/projects/${project.id}/diagrams`}
                    className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", pathname.includes(`/projects/${project.id}`) ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/30 hover:text-white")}>
                    <span className="text-sm truncate">{project.name}</span>
                  </Link>
                ))
              )}
            </div>
          </div>
        )}

        {/* Projects (collapsed) */}
        {collapsed && (
          <div className="pt-6 space-y-1">
            <div className="flex items-center justify-center px-2 pb-2">
              <FolderOpen className="h-5 w-5 text-gray-500" aria-hidden="true" />
            </div>

            {(projectsQ.isLoading ? [] : projects).slice(0, 3).map((project) => (
              <Link
                key={project.id}
                href={`/projects/${project.id}/diagrams`}
                title={project.name} // native tooltip
                aria-label={project.name}
                className={cn(
                  "group flex items-center justify-center rounded-lg transition-all",
                  "h-10 w-10 mx-auto",
                  pathname.includes(`/projects/${project.id}`) ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/30 hover:text-white",
                )}>
                <span className="text-xs font-semibold">{project.name?.trim().split(/\s+/)[0] ?? "?"}</span>
              </Link>
            ))}
          </div>
        )}
      </nav>

      {/* Bottom */}
      <div className="border-t border-slate-800/50 p-3 space-y-1">
        {bottomItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href;

          return (
            <Link
              key={item.href}
              href={item.href}
              aria-label={collapsed ? item.label : undefined}
              className={cn("flex items-center gap-3 px-3 py-2 rounded-lg transition-all", isActive ? "bg-slate-800/50 text-white" : "text-gray-400 hover:bg-slate-800/50 hover:text-white", collapsed && "justify-center")}>
              <Icon className="w-5 h-5 flex-shrink-0" aria-hidden="true" />
              {!collapsed && <span className="text-sm">{item.label}</span>}
            </Link>
          );
        })}

        <button
          type="button"
          onClick={handleLogout}
          aria-label={collapsed ? "Sign out" : undefined}
          className={cn("w-full flex items-center gap-3 px-3 py-2 rounded-lg transition-all text-gray-400 hover:bg-slate-800/50 hover:text-white", collapsed && "justify-center")}>
          <LogOut className="w-5 h-5 flex-shrink-0" aria-hidden="true" />
          {!collapsed && <span className="text-sm">Sign out</span>}
        </button>

        {/* Signed-in user */}
        <div className={cn("flex items-center gap-3 px-3 py-2 mt-2", collapsed && "justify-center")} title={collapsed ? user?.email : undefined}>
          <div className="w-8 h-8 rounded-full bg-teal-500 flex items-center justify-center text-white font-semibold text-sm flex-shrink-0" aria-hidden="true">
            {getUserInitials(user?.displayName, user?.email)}
          </div>

          {!collapsed && (
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-white truncate">{user?.displayName || "User"}</p>
              <p className="text-xs text-gray-500 truncate">{user?.email || ""}</p>
            </div>
          )}
        </div>
      </div>
    </aside>
    </>
  );
}
