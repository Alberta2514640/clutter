import Image from "next/image";
import { Button } from "@/components/ui/button";
import Link from "next/link";

type NavbarProps = {
  showLogin?: boolean; // prop to hide/show login button
};

export default function Navbar({ showLogin = true }: NavbarProps) {
  return (
    <header className="fixed top-0 inset-x-0 z-50 border-b border-slate-800/50 bg-black/40 backdrop-blur-xl supports-[backdrop-filter]:bg-black/30">
      <div className="container mx-auto py-1 flex items-center justify-between">
        {/* Left: logo */}
        <Link href="/" className="flex items-center gap-3">
          <Image src="/logos/logo_text.png" alt="Clutter logo" width={90} height={120} className="object-contain" />
        </Link>

        {/* Right: Login button (hidden on login page) */}
        {showLogin && (
          <div className="flex items-center gap-6">
            <Button asChild className="bg-cyan-600 hover:bg-cyan-700 px-8">
              <Link href="/login">Login</Link>
            </Button>
          </div>
        )}
      </div>
    </header>
  );
}
