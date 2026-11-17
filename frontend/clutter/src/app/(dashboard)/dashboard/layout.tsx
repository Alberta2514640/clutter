import DashboardSidebar from "@/components/common/Sidebar";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white md:flex-row">
      {/* Sidebar */}
      <DashboardSidebar />

      <main className="min-h-0 min-w-0 flex-1 overflow-y-auto">
        <div className="container mx-auto">{children}</div>
      </main>
    </div>
  );
}
