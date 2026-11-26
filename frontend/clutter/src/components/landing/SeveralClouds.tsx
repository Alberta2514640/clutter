import { DotGrid } from "@/components/common/CanvasMotif";
import CloudLogo from "@/components/common/CloudLogo";
import { Archive, Globe, HardDrive, Server, Shield, Table2, Zap } from "lucide-react";
import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";

type Block = { icon: typeof Server; kind: string; name: string };

// One deployment split across two clouds, as the real-cloud test ran it
// (docs/reports/2026-11-20-real-cloud-verification.pdf, pages 3 and 4): the
// network and everything in it on OpenStack, the rest on AWS. Names are the
// Full app example's (backend/adapter/internal/examples/6-full-app.json).
const inNetwork: Block[] = [
  { icon: Server, kind: "VM", name: "web" },
  { icon: Server, kind: "VM", name: "db" },
  { icon: Shield, kind: "Firewall", name: "web-fw" },
  { icon: Shield, kind: "Firewall", name: "db-fw" },
  { icon: HardDrive, kind: "Volume", name: "db-data" },
];
const onAws: Block[] = [
  { icon: Globe, kind: "HTTP API", name: "orders-api" },
  { icon: Zap, kind: "Function", name: "orders" },
  { icon: Table2, kind: "Table", name: "orders-table" },
  { icon: Archive, kind: "Bucket", name: "files" },
];

function BlockList({ blocks }: { blocks: Block[] }) {
  return (
    <ul className="grid grid-cols-2 gap-2">
      {blocks.map(({ icon: Icon, kind, name }) => (
        <li key={name} className="surface-card flex min-w-0 items-center gap-2 rounded-md px-2.5 py-2">
          <Icon className="text-link size-4 shrink-0" aria-hidden="true" />
          <span className="min-w-0 leading-tight">
            <span className="block truncate font-mono text-[13px]">{name}</span>
            <span className="text-muted-foreground block text-[11px]">{kind}</span>
          </span>
        </li>
      ))}
    </ul>
  );
}

function CloudLane({ connector, title, children }: { connector: string; title: string; children: React.ReactNode }) {
  return (
    <div className="surface-card relative p-4">
      <h3 className="mb-3 flex items-center gap-2 text-sm font-semibold">
        <CloudLogo connector={connector} size={20} />
        {title}
      </h3>
      {children}
    </div>
  );
}

export function SeveralClouds() {
  return (
    <section id="several-clouds" aria-labelledby="several-title" className="border-border scroll-mt-20 border-t py-20 md:py-28">
      <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 md:px-8 lg:grid-cols-[5fr_7fr] lg:gap-16">
        <div className={motion.reveal}>
          <Kicker>Several clouds</Kicker>
          <SectionHeading id="several-title">One diagram, several clouds. Even in one deployment.</SectionHeading>
          <p className="text-muted-foreground mt-5 text-lg leading-relaxed">
            Give a diagram a deployment on each cloud you connect, each with its own settings. Or split one deployment: a block one cloud cannot build runs on another you
            connected.
          </p>
        </div>

        <figure className={`border-border bg-muted/40 relative overflow-hidden rounded-2xl border p-4 sm:p-6 ${motion.reveal}`}>
          <DotGrid fade="none" />
          <div className="relative grid gap-4 sm:grid-cols-2">
            <CloudLane connector="general-openstack" title="On OpenStack">
              <div className="border-primary/60 rounded-lg border border-dashed p-2.5">
                <p className="text-muted-foreground mb-2 font-mono text-[11px]">app-net · Network</p>
                <BlockList blocks={inNetwork} />
              </div>
            </CloudLane>
            <CloudLane connector="aws" title="On AWS">
              <BlockList blocks={onAws} />
            </CloudLane>
          </div>
          <figcaption className="text-muted-foreground relative mt-4 text-sm">
            One deployment across two clouds: 10 blocks, 48 cloud resources.
          </figcaption>
        </figure>
      </div>
    </section>
  );
}
