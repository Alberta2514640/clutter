import { kindIconClass } from "@/lib/features/catalog/display";
import { Archive, Globe, HardDrive, Network, Server, Shield, Table2, Zap } from "lucide-react";
import { DemoClip } from "./DemoClip";
import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";

// B's eight building blocks, with the icons and colours B's editor gives
// them (KindIcon.tsx, KIND_TINT).
const blocks = [
  { kind: "vm", icon: Server, name: "VM" },
  { kind: "network", icon: Network, name: "Network" },
  { kind: "firewall", icon: Shield, name: "Firewall" },
  { kind: "volume", icon: HardDrive, name: "Volume" },
  { kind: "bucket", icon: Archive, name: "Bucket" },
  { kind: "function", icon: Zap, name: "Function" },
  { kind: "table", icon: Table2, name: "Table" },
  { kind: "http_api", icon: Globe, name: "HTTP API" },
];

// Draw once: the blocks, and the demo clip ("Watch the demo" lands here).
export function DrawOnce() {
  return (
    <section id="demo" aria-labelledby="draw-title" className="border-border scroll-mt-20 border-t py-20 md:py-28">
      <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 md:px-8 lg:grid-cols-[5fr_7fr]">
        <div className={motion.reveal}>
          <Kicker>Draw</Kicker>
          <SectionHeading id="draw-title">Draw it once, in plain building blocks.</SectionHeading>
          <p className="text-muted-foreground mt-5 text-lg leading-relaxed">
            Click or drag a block onto the canvas, give it a name, and link it to the others. Each block has a short form with plain choices, such as a small, medium or large VM.
          </p>
          <ul className="mt-8 grid grid-cols-2 gap-2 sm:grid-cols-4 lg:grid-cols-2 xl:grid-cols-4">
            {blocks.map(({ kind, icon: Icon, name }) => (
              <li key={name} className="surface-card flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium">
                <Icon className={`${kindIconClass(kind)} size-4 shrink-0`} aria-hidden="true" />
                {name}
              </li>
            ))}
          </ul>
        </div>
        <figure className={motion.reveal}>
          <DemoClip describedBy="demo-caption" />
          <figcaption id="demo-caption" className="text-muted-foreground mt-3 text-sm">
            Add a VM, name it, drop it into a network, then deploy until What was built appears.
          </figcaption>
        </figure>
      </div>
    </section>
  );
}
