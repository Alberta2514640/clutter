import type { Metadata } from "next";

export const metadata: Metadata = { title: "Diagrams" };

export default function ProjectDiagramsLayout({ children }: { children: React.ReactNode }) {
  return children;
}
