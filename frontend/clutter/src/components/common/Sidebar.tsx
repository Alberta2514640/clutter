'use client';

import { Button } from '@/components/ui/button';
import { useProjectState } from '@/lib/stores/projectStore';
import { useUserState } from '@/lib/stores/userStore';
import { cn } from '@/lib/utils';
import {
  BarChart3,
  BookTemplate,
  ChevronLeft,
  ChevronRight,
  FolderOpen,
  HelpCircle,
  LayoutDashboard,
  Plus,
  Settings,
  Sparkles,
} from 'lucide-react';
import Image from 'next/image';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { useState } from 'react';

interface SidebarProps {
  className?: string;
}

export default function DashboardSidebar({ className }: SidebarProps) {
  const [collapsed, setCollapsed] = useState(true);
  const pathname = usePathname();

  // Get data from stores
  const { projects } = useProjectState();
  const { user } = useUserState();

  const navItems = [
    { icon: LayoutDashboard, label: 'Overview', href: '/dashboard' }
  ];

  const bottomItems = [
    { icon: Settings, label: 'Admin Panel', href: '/settings/organization/manage' },
    { icon: BookTemplate, label: 'Templates', href: '/templates' },
    { icon: BarChart3, label: 'Insights', href: '/insights' },
    { icon: HelpCircle, label: 'Help', href: '/help' },
    { icon: Sparkles, label: "What's New", href: '/whats-new' },
  ];

  const handleExpand = () => {
    if (collapsed) {
      setCollapsed(false);
    }
  };

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation();
    setCollapsed(!collapsed);
  };

  // Helper to get project emoji/icon (you can enhance this)
  const getProjectIcon = (name: string) => {
    if (name.toLowerCase().includes('web')) return '🌐';
    if (name.toLowerCase().includes('data')) return '📊';
    if (name.toLowerCase().includes('monitor')) return '📈';
    if (name.toLowerCase().includes('api')) return '🔌';
    if (name.toLowerCase().includes('mobile')) return '📱';
    return '📁'; // default
  };

  // Helper to get user initials
  const getUserInitials = (displayName?: string, email?: string) => {
    if (displayName) {
      const names = displayName.split(' ');
      if (names.length >= 2) {
        return `${names[0][0]}${names[1][0]}`.toUpperCase();
      }
      return displayName.slice(0, 2).toUpperCase();
    }
    if (email) {
      return email.slice(0, 2).toUpperCase();
    }
    return '??';
  };

  return (
    <aside
      onClick={handleExpand}
      className={cn(
        'relative flex flex-col border-r border-slate-800/50 bg-slate-900/40 backdrop-blur-xl transition-all duration-300',
        collapsed ? 'w-16 cursor-pointer' : 'w-64',
        className
      )}
    >
      {/* Expand Button - when collapsed */}
      {collapsed && (
        <button
          onClick={handleToggle}
          className="absolute -right-3 top-1/2 z-10 p-1.5 rounded-full bg-slate-800 border border-slate-700 hover:bg-slate-700 transition-colors shadow-lg">
          <ChevronRight className="w-4 h-4 text-gray-400" />
        </button>
      )}
      {/* Collapse Button - when expanded */}
      {!collapsed && (
        <button
          onClick={handleToggle}
          className="absolute -right-3 top-1/2 z-10 p-1.5 rounded-full bg-slate-800 border border-slate-700 hover:bg-slate-700 transition-colors shadow-lg"
        >
          <ChevronLeft className="w-4 h-4 text-gray-400" />
        </button>
      )}

      {/* Logo Section */}
      <div className="flex items-center justify-center p-4 border-b border-slate-800/50">
        {collapsed ? (
          <Image
            src="/logos/logo.png"
            alt="Clutter"
            width={80}
            height={80}
            className="object-contain"
          />
        ) : (
          <Image
            src="/logos/logo_text.png"
            alt="Clutter"
            width={80}
            height={80}
            className="object-contain"
          />
        )}
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
              className={cn(
                'flex items-center gap-3 px-3 py-2 rounded-lg transition-all',
                isActive 
                  ? 'bg-teal-500/10 text-teal-400' 
                  : 'text-gray-400 hover:bg-slate-800/50 hover:text-white',
                collapsed && 'justify-center'
              )}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && <span className="text-sm font-medium">{item.label}</span>}
            </Link>
          );
        })}

        {/* Projects Section - EXPANDED */}
        {!collapsed && (
          <div className="pt-6" onClick={(e) => e.stopPropagation()}>
            <div className="flex items-center justify-between px-3 pb-2">
              <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                Projects
              </span>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 w-6 p-0 hover:bg-slate-800/50"
                asChild
              >
                <Link href="/projects/new">
                  <Plus className="w-4 h-4 text-gray-400" />
                </Link>
              </Button>
            </div>
            
            <div className="space-y-1">
              {projects.length === 0 ? (
                <div className="px-3 py-2 text-xs text-gray-500 text-center">
                  No projects yet
                </div>
              ) : (
                projects.map((project) => (
                  <Link
                    key={project.projectId}
                    href={`/projects/${project.projectId}/diagrams`}
                    className={cn(
                      'flex items-center gap-3 px-3 py-2 rounded-lg transition-all',
                      pathname.includes(`/projects/${project.projectId}`)
                        ? 'bg-slate-800/50 text-white'
                        : 'text-gray-400 hover:bg-slate-800/30 hover:text-white'
                    )}
                  >
                    <span className="text-lg flex-shrink-0">
                      {getProjectIcon(project.name)}
                    </span>
                    <span className="text-sm truncate">{project.name}</span>
                  </Link>
                ))
              )}
            </div>
          </div>
        )}

        {/* Projects Section - COLLAPSED */}
        {collapsed && (
          <div className="pt-6 space-y-1">
            <div className="flex items-center justify-center px-2 pb-2">
              <FolderOpen className="text-gray-500 px-0.5" />
            </div>
            {projects.slice(0, 3).map((project) => (
              <Link
                key={project.projectId}
                href={`/projects/${project.projectId}/diagrams`}
                onClick={(e) => e.stopPropagation()}
                className={cn(
                  'flex items-center justify-center px-3 py-2 rounded-lg transition-all',
                  pathname.includes(`/projects/${project.projectId}`)
                    ? 'bg-slate-800/50 text-white'
                    : 'text-gray-400 hover:bg-slate-800/30 hover:text-white'
                )}
              >
                <span className="text-lg">{getProjectIcon(project.name)}</span>
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
              className={cn(
                'flex items-center gap-3 px-3 py-2 rounded-lg transition-all',
                isActive 
                  ? 'bg-slate-800/50 text-white' 
                  : 'text-gray-400 hover:bg-slate-800/50 hover:text-white',
                collapsed && 'justify-center'
              )}
            >
              <Icon className="w-5 h-5 flex-shrink-0" />
              {!collapsed && <span className="text-sm">{item.label}</span>}
            </Link>
          );
        })}

        {/* User Profile */}
        <div 
          onClick={(e) => e.stopPropagation()}
          className={cn(
            'flex items-center gap-3 px-3 py-2 mt-2 rounded-lg hover:bg-slate-800/50 cursor-pointer transition-all',
            collapsed && 'justify-center'
          )}
        >
          <div className="w-8 h-8 rounded-full bg-teal-500 flex items-center justify-center text-white font-semibold text-sm flex-shrink-0">
            {getUserInitials(user?.displayName, user?.email)}
          </div>
          {!collapsed && user && (
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-white truncate">
                {user.displayName || 'User'}
              </p>
              <p className="text-xs text-gray-500 truncate">
                {user.email || ''}
              </p>
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}