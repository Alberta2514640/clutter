// /src/app/(settings)/layout.tsx
import DashboardSidebar from "@/components/common/Sidebar";

//same sidebar layout as dashboard
export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex h-screen flex-col overflow-hidden bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white md:flex-row">
      {/* Sidebar */}
      <DashboardSidebar />

      <main className="min-h-0 min-w-0 flex-1 overflow-y-auto">
        <div className="container mx-auto px-4 py-6 md:px-8">{children}</div>
      </main>
    </div>
  );
}
