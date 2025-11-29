// app/layout.tsx

import "./globals.css";
import Script from "next/script";
import { QueryProvider } from "@/providers/QueryProvider";
import { AuthProvider } from "@/providers/AuthProvider";

export const metadata = { title: "Clutter" };

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        {/* Google Identity Services Script */}
        <Script src="https://accounts.google.com/gsi/client" strategy="afterInteractive" />
      </head>

      <body>
        <AuthProvider>
          <QueryProvider>{children}</QueryProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
