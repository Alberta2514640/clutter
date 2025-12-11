import "@xyflow/react/dist/style.css";
import type { ReactNode } from "react";

export default function DiagramLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-screen w-screen overflow-hidden text-white">
      {/* Background similar to your screenshot */}
      <div className="h-full w-full bg-[radial-gradient(1200px_800px_at_30%_20%,rgba(40,60,90,0.55),transparent_60%),radial-gradient(900px_700px_at_85%_80%,rgba(0,140,120,0.35),transparent_55%),linear-gradient(180deg,#05070d_0%,#061021_45%,#061b1b_100%)]">
        {children}
      </div>
    </div>
  );
}