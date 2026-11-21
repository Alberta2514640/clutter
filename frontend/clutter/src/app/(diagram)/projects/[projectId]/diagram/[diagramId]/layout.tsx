import type { Metadata } from "next";

export const metadata: Metadata = { title: "Diagram" };

export default function DiagramLayout({ children }: { children: React.ReactNode }) {
  return children;
}
