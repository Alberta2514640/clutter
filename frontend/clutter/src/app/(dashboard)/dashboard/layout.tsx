import Navbar from "@/components/common/Navbar";

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      <Navbar />
      <main className="container mx-auto">
        {children}
      </main>
      {/* Background decorations */}
    </div>
  );
}