import Image from "next/image";
import { Button } from "@/components/ui/button";
import Link from "next/link";

export default function Navbar() {
  return (
    <header className="top-0 left-0 right-0 z-50 w-full border-b border-slate-800/50 bg-black/20 backdrop-blur-md">
      <div className="max-w-7xl mx-auto px-10 py-4 flex items-center justify-between">
        {/* Left: logo */}
        <div className="flex items-center gap-3">
          <Link href="/" className="flex items-center gap-3">
            <Image
              src="/logos/logo_text.png"
              alt="Clutter logo"
              width={100}
              height={100}
              className="object-contain"
            />
          </Link>
        </div>

        {/* Right: Contact us and Login button */}
        <div className="flex items-center gap-6">
          <Link
            href="/contact"
            className="text-gray-300 hover:text-white transition-colors"
          >
            Contact us
          </Link>
          <Button asChild className="bg-teal-500 hover:bg-teal-600">
            <Link href="/login">Login</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}
