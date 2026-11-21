import GoogleAuthProvider from "@/providers/GoogleAuthProvider";
import type { Metadata } from "next";

export const metadata: Metadata = { title: "Sign in" };

// Google's sign-in script loads only here, the one page that uses it, so the
// landing page and the signed-in app do not download it.
export default function LoginLayout({ children }: { children: React.ReactNode }) {
  return <GoogleAuthProvider>{children}</GoogleAuthProvider>;
}
