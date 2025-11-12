import Navbar from '@/components/common/Navbar';
import Link from 'next/link';

export default function Home() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
      {/* Navigation */}
      <Navbar />

      {/* Hero Section */}
      <main className="flex flex-col items-center justify-center min-h-screen px-6">
        <div className="max-w-5xl mx-auto text-center space-y-12">
          {/* Main Heading */}
          <h1 className="text-6xl md:text-7xl lg:text-8xl font-bold tracking-tight">
            Infrastructure, simplified
          </h1>

          {/* CTA Button */}
          <div className="pt-8">
            <Link
              href="/login"
              className="inline-block px-12 py-5 bg-teal-500 hover:bg-teal-600 text-white text-xl font-semibold rounded-2xl transition-all transform hover:scale-105 shadow-lg hover:shadow-teal-500/50"
            >
              Start Here
            </Link>
          </div>
        </div>
      </main>

      {/* Background Decoration */}
      <div className="fixed inset-0 pointer-events-none opacity-30">
        <div className="absolute top-1/4 right-1/4 w-96 h-96 bg-teal-500 rounded-full filter blur-3xl"></div>
        <div className="absolute bottom-1/4 left-1/4 w-96 h-96 bg-blue-500 rounded-full filter blur-3xl"></div>
      </div>
    </div>
  );
}