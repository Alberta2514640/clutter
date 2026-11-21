import "@xyflow/react/dist/style.css";
import type { ReactNode } from "react";

// The editor is full screen, outside the app shell. B's editor follows the
// viewer's theme; A's older editor sets its own dark scope (EditorSwitch).
export default function DiagramLayout({ children }: { children: ReactNode }) {
  return <div className="bg-background text-foreground h-dvh w-screen overflow-hidden">{children}</div>;
}
