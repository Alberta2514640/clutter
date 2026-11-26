"use client";

import ClutterLogo from "@/components/common/ClutterLogo";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { PROJECTS_PATH } from "@/lib/features/organization/home";
import { useCurrentOrganization } from "@/lib/features/organization/hooks";
import { sortOrganizations } from "@/lib/features/organization/sort";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { useLogout, useMe } from "@/lib/features/user/hooks";
import { tintBadgeClass, tintForId } from "@/lib/tint";
import { cn } from "@/lib/utils";
import { Cable, ChevronsUpDown, FolderKanban, LogOut, Menu, Plus, Settings, UserRound, type LucideIcon } from "lucide-react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import ErrorState from "./ErrorState";
import { isThemeChoice } from "./theme";
import { useTheme } from "./ThemeProvider";
import ThemeScope from "./ThemeScope";
import { THEME_OPTIONS } from "./ThemeToggle";

// The app's sidebar, always dark. Top to bottom: the team switcher, Projects,
// Connections, Settings, and the "You" menu. Below md it is a drawer behind
// the top bar's menu button. The UI says "team"; code, the API and URLs keep
// "organization".

const CONNECTIONS_HREF = "/settings/organization/connections";
const SETTINGS_HREF = "/settings/organization/manage";
const NEW_TEAM_HREF = "/settings/organization/create-org";
const ACCOUNT_HREF = "/settings/account";

type NavItem = { label: string; href: string; icon: LucideIcon; isActive: (pathname: string) => boolean };

const NAV_ITEMS: NavItem[] = [
  {
    label: "Projects",
    href: PROJECTS_PATH,
    icon: FolderKanban,
    isActive: (p) => p === PROJECTS_PATH || p.startsWith("/projects"),
  },
  {
    label: "Connections",
    href: CONNECTIONS_HREF,
    icon: Cable,
    // The AWS account page redirects here; until it does, it belongs here too.
    isActive: (p) => p.startsWith(CONNECTIONS_HREF) || p.startsWith("/settings/organization/aws-account"),
  },
  { label: "Settings", href: SETTINGS_HREF, icon: Settings, isActive: (p) => p.startsWith(SETTINGS_HREF) },
];

const initialsOf = (displayName?: string, email?: string) => {
  const name = displayName?.trim();
  if (name) {
    const parts = name.split(/\s+/);
    return (parts.length >= 2 ? `${parts[0][0]}${parts[1][0]}` : name.slice(0, 2)).toUpperCase();
  }
  return email ? email.slice(0, 2).toUpperCase() : "?";
};

const focusRing = "outline-none focus-visible:ring-[3px] focus-visible:ring-sidebar-ring/50";

// A team's initial on a tint chosen from its id, so each team keeps its colour.
function TeamMark({ id, name, className }: { id: string; name: string; className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        tintBadgeClass(tintForId(id)),
        "flex size-7 shrink-0 items-center justify-center rounded-md text-xs font-semibold",
        className,
      )}>
      {name.trim().slice(0, 1).toUpperCase() || "?"}
    </span>
  );
}

function TeamSwitcher({ onNavigate }: { onNavigate?: () => void }) {
  const router = useRouter();
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const current = useCurrentOrganization(token);
  const teams = useMemo(() => sortOrganizations(current.organizations), [current.organizations]);

  if (meQ.isLoading || current.isLoading) {
    return (
      <div role="status">
        <span className="sr-only">Loading your teams…</span>
        <Skeleton className="h-12 w-full rounded-lg" />
      </div>
    );
  }
  if (current.isError) {
    return (
      <ErrorState
        variant="inline"
        description="Could not load your teams."
        onRetry={() => current.refetch()}
        retrying={current.isFetching}
        className="px-1"
      />
    );
  }
  if (!current.isSuccess) return null;
  if (!current.organization) {
    return (
      <Button asChild variant="outline" className="w-full justify-start">
        <Link href={NEW_TEAM_HREF} onClick={onNavigate}>
          <Plus aria-hidden="true" />
          Create a team
        </Link>
      </Button>
    );
  }

  const name = current.organization.name;
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="team-switcher"
          className={cn(
            "border-sidebar-border hover:bg-sidebar-accent data-[state=open]:bg-sidebar-accent flex w-full items-center gap-3 rounded-lg border p-2 text-left transition-colors",
            focusRing,
          )}>
          <TeamMark id={current.organization.id} name={name} />
          <span className="min-w-0 flex-1">
            <span className="text-muted-foreground block text-xs">Team</span>
            <span className="block truncate text-sm font-medium">{name}</span>
          </span>
          <ChevronsUpDown className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-(--radix-dropdown-menu-trigger-width) min-w-56">
        <DropdownMenuLabel>Teams</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={current.organizationId ?? undefined}
          onValueChange={(organizationId) => {
            // A team's work starts on its Projects page.
            current.select(organizationId);
            onNavigate?.();
            router.push(PROJECTS_PATH);
          }}>
          {teams.map((team) => (
            <DropdownMenuRadioItem key={team.id} value={team.id}>
              <TeamMark id={team.id} name={team.name} className="size-6" />
              <span className="truncate">{team.name}</span>
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link href={NEW_TEAM_HREF} onClick={onNavigate}>
            <Plus aria-hidden="true" />
            New team
          </Link>
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function UserMenu({ onNavigate }: { onNavigate?: () => void }) {
  const router = useRouter();
  const meQ = useMe();
  const logout = useLogout();
  const forgetUserData = usePreferencesStore((s) => s.forgetUserData);
  const { theme, setTheme } = useTheme();
  const user = meQ.data ?? null;

  if (!user) {
    return meQ.isLoading ? (
      <div role="status">
        <span className="sr-only">Loading your account…</span>
        <Skeleton className="h-12 w-full rounded-lg" />
      </div>
    ) : null;
  }

  const name = user.displayName || user.email || "You";
  // The toaster lives in the root providers, so the toast shows on the sign-in page.
  // The next person in this browser gets none of this person's team memory;
  // the theme stays.
  const signOut = () => {
    logout();
    forgetUserData();
    toast.success("You are signed out.");
    router.push("/login");
    router.refresh();
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          data-testid="user-menu"
          className={cn(
            "hover:bg-sidebar-accent data-[state=open]:bg-sidebar-accent flex w-full items-center gap-3 rounded-lg p-2 text-left transition-colors",
            focusRing,
          )}>
          <Avatar className="size-8">
            {user.pictureUrl && <AvatarImage src={user.pictureUrl} alt="" referrerPolicy="no-referrer" />}
            <AvatarFallback className="bg-sidebar-primary/20 text-sidebar-primary text-xs font-semibold">
              {initialsOf(user.displayName, user.email)}
            </AvatarFallback>
          </Avatar>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-medium">{name}</span>
            {user.email && <span className="text-muted-foreground block truncate text-xs">{user.email}</span>}
          </span>
          <ChevronsUpDown className="text-muted-foreground size-4 shrink-0" aria-hidden="true" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-(--radix-dropdown-menu-trigger-width) min-w-56">
        <DropdownMenuLabel>You</DropdownMenuLabel>
        <DropdownMenuItem asChild>
          <Link href={ACCOUNT_HREF} onClick={onNavigate}>
            <UserRound aria-hidden="true" />
            Account and preferences
          </Link>
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuLabel>Theme</DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(value) => {
            if (isThemeChoice(value)) setTheme(value);
          }}>
          {THEME_OPTIONS.map(({ value, label, icon: Icon }) => (
            // The menu stays open, so the change can be seen and undone.
            <DropdownMenuRadioItem key={value} value={value} onSelect={(event) => event.preventDefault()}>
              <Icon aria-hidden="true" />
              {label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={signOut}>
          <LogOut aria-hidden="true" />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function SidebarBody({ onNavigate }: { onNavigate?: () => void }) {
  const pathname = usePathname() ?? "";
  return (
    <div className="bg-sidebar text-sidebar-foreground flex h-full flex-col">
      <div className="flex h-14 shrink-0 items-center px-4">
        <Link href={PROJECTS_PATH} onClick={onNavigate} aria-label="Clutter, go to Projects" className={cn("group rounded-md", focusRing)}>
          <ClutterLogo size={24} withName motion="lift" />
        </Link>
      </div>
      <div className="px-3">
        <TeamSwitcher onNavigate={onNavigate} />
      </div>
      <nav aria-label="Main" className="mt-6 flex-1 space-y-0.5 overflow-y-auto px-3">
        {NAV_ITEMS.map(({ label, href, icon: Icon, isActive }) => {
          const active = isActive(pathname);
          return (
            <Link
              key={href}
              href={href}
              onClick={onNavigate}
              aria-current={active ? "page" : undefined}
              className={cn(
                "group/nav text-sidebar-foreground hover:bg-sidebar-accent hover:text-sidebar-accent-foreground aria-[current=page]:bg-sidebar-accent aria-[current=page]:text-sidebar-accent-foreground flex h-8 items-center gap-2.5 rounded-md px-2.5 text-[13px] font-medium transition-colors",
                focusRing,
              )}>
              <Icon
                className="text-muted-foreground group-aria-[current=page]/nav:text-sidebar-primary size-4 shrink-0 transition-colors"
                aria-hidden="true"
              />
              {label}
            </Link>
          );
        })}
      </nav>
      <div className="border-sidebar-border border-t p-3">
        <UserMenu onNavigate={onNavigate} />
      </div>
    </div>
  );
}

// The sidebar from md up: sticky, full height.
export default function AppSidebar() {
  return (
    <ThemeScope theme="dark" className="border-sidebar-border sticky top-0 hidden h-dvh w-60 shrink-0 border-r md:block">
      <SidebarBody />
    </ThemeScope>
  );
}

// Below md: a top bar with the menu button, and the sidebar as a drawer.
export function MobileNav() {
  const pathname = usePathname();
  // The drawer stays open only on the page it was opened on, so any
  // navigation closes it.
  const [openOn, setOpenOn] = useState<string | null>(null);
  const open = openOn !== null && openOn === pathname;
  const close = () => setOpenOn(null);

  return (
    <ThemeScope
      theme="dark"
      className="border-sidebar-border bg-sidebar text-sidebar-foreground sticky top-0 z-40 flex h-14 shrink-0 items-center gap-2 border-b px-2 md:hidden">
      <Sheet open={open} onOpenChange={(next) => setOpenOn(next ? pathname : null)}>
        <SheetTrigger asChild>
          <Button variant="ghost" size="icon" aria-label="Open menu" className="hover:bg-sidebar-accent text-sidebar-foreground">
            <Menu className="size-5" aria-hidden="true" />
          </Button>
        </SheetTrigger>
        <SheetContent side="left" closeLabel="Close menu" aria-describedby={undefined} className="border-sidebar-border w-72 gap-0 p-0">
          <SheetTitle className="sr-only">Menu</SheetTitle>
          <SidebarBody onNavigate={close} />
        </SheetContent>
      </Sheet>
      <Link href={PROJECTS_PATH} aria-label="Clutter, go to Projects" className={cn("group rounded-md", focusRing)}>
        <ClutterLogo size={24} withName motion="lift" />
      </Link>
    </ThemeScope>
  );
}
