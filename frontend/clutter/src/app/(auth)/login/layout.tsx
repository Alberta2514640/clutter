import GoogleAuthProvider from "@/providers/GoogleAuthProvider";

// Google's sign-in script loads only here, the one page that uses it, so the
// landing page and the signed-in app do not download it.
export default function LoginLayout({ children }: { children: React.ReactNode }) {
  return <GoogleAuthProvider>{children}</GoogleAuthProvider>;
}
