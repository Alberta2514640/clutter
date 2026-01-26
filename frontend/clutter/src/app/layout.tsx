// app/layout.tsx

import Script from "next/script";
import GoogleAuthProvider from "../providers/GoogleAuthProvider";
import "./globals.css";
import { Providers } from "./providers";

// You can add metadata here if you want
export const metadata = {
  title: "Clutter",
  description: "Cloud IaC Diagram Builder",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <head>
        {/* Google Identity Services Script (Required for GSI login) */}
        <Script src="https://accounts.google.com/gsi/client" strategy="afterInteractive" />
      </head>

      <body>
        {/* <GoogleAuthProvider>{children}</GoogleAuthProvider> */}
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
