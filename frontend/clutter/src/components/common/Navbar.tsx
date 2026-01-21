import Image from "next/image";
import { Button } from "@/components/ui/button";
import Link from "next/link";

type NavbarProps = {
  showLogin?: boolean;
};

export default function Navbar({ showLogin = true }: NavbarProps) {
  return (
    <header
      className="
        fixed top-0 inset-x-0 z-50
        border-b border-white/3
        bg-gradient-to-b from-slate-950/35 to-slate-950/10
        backdrop-blur-xl
        shadow-[0_1px_0_rgba(255,255,255,0.03)]
        supports-[backdrop-filter]:bg-slate-950/30
      ">
      <div className="container mx-auto flex items-center justify-between px-1 py-0">
        {/* Left: logo */}
        <Link href="/" className="flex items-center gap-3">
          <Image src="/logos/logo_text.png" alt="Clutter logo" width={88} height={28} className="object-contain" priority />
        </Link>

        {/* Right: Login button */}
        {showLogin && (
          <Button
            asChild
            className="
    relative
    rounded-half
    px-8 py-5
    text-cyan-300
    text-lg font-light
    border border-cyan-400/40
    bg-transparent
    shadow-[0_0_0_1px_rgba(34,211,238,0.35),0_0_24px_rgba(34,211,238,0.25)]
    hover:text-cyan-200
    hover:border-cyan-300/70
    hover:shadow-[0_0_0_1px_rgba(34,211,238,0.5),0_0_32px_rgba(34,211,238,0.35)]
    transition-all duration-300
  ">
            <Link href="/login">Login</Link>
          </Button>
        )}
      </div>
    </header>
  );
}
