import AppShell from "@/components/common/AppShell";

// Projects and everything inside a project, in the app shell. One layout for
// the group, so moving between Projects and a project keeps the sidebar.
export default function DashboardGroupLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}
