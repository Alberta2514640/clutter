import Navbar from "@/components/common/Navbar";
import DashboardSidebar from "@/components/common/Sidebar";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white flex">
      {/* Sidebar */}
      <DashboardSidebar />

      <main className="container mx-auto">
        {children}
      </main>
    </div>
  );
}