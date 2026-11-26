import AppShell from "@/components/common/AppShell";

// Connections, team settings and the viewer's own account, in the app shell.
export default function SettingsGroupLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}
