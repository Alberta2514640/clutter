import Image from "next/image";
import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function Navbar() {
  return (
    <header className="w-full border-b border-slate-800/50 bg-black/20 backdrop-blur-md">
      <div className="container mx-auto px-6 py-4 flex items-center justify-between">
        {/* Left: logo */}
        <Link href="/" className="flex items-center gap-3">
          <Image
            src="/logos/logo_text.png"
            alt="Clutter logo"
            width={100}
            height={100}
            className="object-contain"
          />
        </Link>

        {/* Right: Login button */}
        <div className="flex items-center gap-6">
          <Button asChild className="bg-teal-500 hover:bg-teal-600 px-8">
            <Link href="/login">Login</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
