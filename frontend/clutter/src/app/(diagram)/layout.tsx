
import DashboardSidebar from "@/components/common/Sidebar";

export default function CanvasLayout({ children }: { children: React.ReactNode }) {
    return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white flex">
      <DashboardSidebar />
      <main className="flex-1 min-w-0 overflow-x-hidden">
        {children}
      </main>
    </div>
  );
}