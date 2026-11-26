import CloudLogo from "@/components/common/CloudLogo";
import { kindBadgeClass, kindIconClass } from "@/lib/features/catalog/display";
import { cn } from "@/lib/utils";
import { Globe, HardDrive, Server, Zap, type LucideIcon } from "lucide-react";
import type { CSSProperties, ReactNode } from "react";

// The hero's living diagram, beside the headline: block tiles and cloud logo
// tiles that float, dashed connectors that draw in once, and a packet that
// travels from a block to its cloud. Decorative (aria-hidden): the blocks and
// clouds are named in text further down. The motion is CSS (the hero-*
// keyframes in globals.css); HeroMotion adds pointer parallax to every
// [data-depth] element and pauses it all off screen.
//
// Each side pairs blocks only with a cloud that builds them (B's manifests):
// an HTTP API and a function on AWS, a VM and a volume on OpenStack.

type Box = { x: number; y: number; w: number; h: number };

// Half a float loop on each axis, in seconds (each runs there and back), and
// how far into the loop the tile starts.
type Float = { y: number; x: number; phase: number };

const CHIP = { w: 112, h: 50 };
const LOGO = { w: 66, h: 66 };

// Geometry of the two clusters, in px inside a 260 x 470 box. The boxes sit
// at a fixed distance from the page centre (at least 24px from the edge), and
// leave the band beside the headline's widest line clear from 1280px up.
const CLUSTER = { w: 260, h: 470, top: 104, fromCentre: 690 };
const LEFT = {
  api: { x: 70, y: 12, ...CHIP },
  jobs: { x: 6, y: 214, ...CHIP },
  aws: { x: 104, y: 392, ...LOGO },
};
const RIGHT = {
  openstack: { x: 150, y: 36, ...LOGO },
  web: { x: 124, y: 220, ...CHIP },
  data: { x: 40, y: 396, ...CHIP },
};

// The packet loop (hero-packet and hero-arrive share it, so the cloud tile
// answers as the packet lands). The two sides run half a loop apart.
const PACKET_LOOP_S = 6;
const LEFT_PACKET_DELAY_S = 1.6;
const RIGHT_PACKET_DELAY_S = LEFT_PACKET_DELAY_S + PACKET_LOOP_S / 2;

const round = (n: number) => Math.round(n * 100) / 100;

// segment is the straight line between two tiles' centres, trimmed to stop
// `gap` px short of each tile's edge, as a start point, a length and an angle.
function segment(a: Box, b: Box, gap = 10) {
  const ax = a.x + a.w / 2;
  const ay = a.y + a.h / 2;
  const dx = b.x + b.w / 2 - ax;
  const dy = b.y + b.h / 2 - ay;
  const len = Math.hypot(dx, dy);
  // The fraction of the centre line that lies inside a tile of this size.
  const inside = (t: Box) => Math.min(dx ? t.w / 2 / Math.abs(dx) : Infinity, dy ? t.h / 2 / Math.abs(dy) : Infinity);
  const t0 = inside(a) + gap / len;
  const t1 = 1 - inside(b) - gap / len;
  return {
    x: round(ax + dx * t0),
    y: round(ay + dy * t0),
    length: round(len * (t1 - t0)),
    angle: round((Math.atan2(dy, dx) * 180) / Math.PI),
  };
}

const floatStyle = (half: number, phase: number): CSSProperties => ({
  animationDuration: `${half}s`,
  animationDelay: `-${phase}s`,
});

type TileProps = {
  float: Float;
  // A static tilt (a rotate-* class).
  tilt: string;
  // What the tile is, shown under it on hover.
  tip: ReactNode;
  tipSide: "left" | "right";
  // Parallax depth in px at the window's edge; none on the narrow layout.
  depth?: number;
  // Position and size: a box on the wide layout, classes on the narrow one.
  box?: Box;
  className?: string;
  children: ReactNode;
};

// A tile floats on two slow loops (y on the outer layer, x and a hair of
// rotation on the inner one, at different lengths, so the path never quite
// repeats), lifts 2px on hover and shows its tip.
function Tile({ float, tilt, tip, tipSide, depth, box, className, children }: TileProps) {
  return (
    <div
      data-depth={depth}
      className={cn("absolute", depth !== undefined && "will-change-transform", className)}
      style={box ? { left: box.x, top: box.y, width: box.w, height: box.h } : undefined}>
      <div className="hero-float-y group relative size-full" style={floatStyle(float.y, float.phase)}>
        <div className={cn("hero-float-x pointer-events-auto size-full", tilt)} style={floatStyle(float.x, float.phase * 1.7)}>
          <div className="surface-card relative flex size-full items-center transition-[translate,border-color] group-hover:-translate-y-0.5 group-hover:[border-color:var(--card-border-hover)]">
            {children}
          </div>
        </div>
        <span
          className={cn(
            "surface-raised text-popover-foreground pointer-events-none absolute top-full mt-2 -translate-y-1 rounded-md border px-2 py-1 text-xs whitespace-nowrap opacity-0 transition-[opacity,translate] group-hover:translate-y-0 group-hover:opacity-100",
            tipSide === "left" ? "left-0" : "right-0",
          )}>
          {tip}
        </span>
      </div>
    </div>
  );
}

function Tip({ name, text }: { name: string; text: string }) {
  return (
    <>
      <span className="font-medium">{name}:</span> <span className="text-muted-foreground">{text}</span>
    </>
  );
}

// A building block as it looks on the canvas.
function BlockFace({ icon: Icon, kind, name, label }: { icon: LucideIcon; kind: string; name: string; label: string }) {
  return (
    <>
      <span className={cn(kindBadgeClass(kind), "ml-2.5 flex size-7 shrink-0 items-center justify-center rounded-md")}>
        <Icon className="size-4" />
      </span>
      <span className="ml-2.5 leading-tight">
        <span className="block text-sm font-medium">{name}</span>
        <span className="text-muted-foreground block text-[11px]">{label}</span>
      </span>
    </>
  );
}

// A cloud's logo, through the one logo registry. `arriveDelay` adds the ring
// that answers a packet landing on this tile.
function LogoFace({ connector, size, arriveDelay }: { connector: string; size: number; arriveDelay?: number }) {
  return (
    <>
      <CloudLogo connector={connector} size={size} className="mx-auto" />
      {arriveDelay !== undefined && (
        <span
          className="hero-arrive border-success-solid pointer-events-none absolute -inset-px rounded-[inherit] border opacity-0"
          style={{ animationDuration: `${PACKET_LOOP_S}s`, animationDelay: `${arriveDelay}s` }}
        />
      )}
    </>
  );
}

type ConnectorProps = {
  from: Box;
  to: Box;
  // The source block's kind, for the line's colour.
  kind: string;
  depth: number;
  drawDelayMs: number;
  // Set for a line that carries a packet, from `from` to `to`.
  packetDelay?: number;
};

// A dashed edge between two tiles in the source block's colour. It draws in
// once (the line slides out of its own clip, so the dashes never stretch).
function Connector({ from, to, kind, depth, drawDelayMs, packetDelay }: ConnectorProps) {
  const s = segment(from, to);
  return (
    <div data-depth={depth} className="absolute will-change-transform" style={{ left: s.x, top: s.y }}>
      <div
        className={cn("absolute -top-1 left-0 h-2", kindIconClass(kind))}
        style={{ width: s.length, transform: `rotate(${s.angle}deg)`, transformOrigin: "0 50%" }}>
        <div className="absolute inset-0 overflow-hidden">
          <div className="hero-draw absolute inset-x-0 top-1/2 border-t border-dashed border-current opacity-40" style={{ animationDelay: `${drawDelayMs}ms` }} />
        </div>
        {packetDelay !== undefined && (
          <div className="hero-packet absolute inset-0 opacity-0" style={{ animationDuration: `${PACKET_LOOP_S}s`, animationDelay: `${packetDelay}s` }}>
            <span className="ring-hero absolute top-1/2 left-0 size-1.5 -translate-1/2 rounded-full bg-current ring-2" />
          </div>
        )}
      </div>
    </div>
  );
}

const clusterBox = (side: "left" | "right"): CSSProperties => ({
  top: CLUSTER.top,
  width: CLUSTER.w,
  height: CLUSTER.h,
  [side]: `max(24px, calc(50% - ${CLUSTER.fromCentre}px))`,
});

const TIPS = {
  api: <Tip name="HTTP API" text="a web address for your code" />,
  jobs: <Tip name="Function" text="code that runs when called" />,
  web: <Tip name="VM" text="a server that runs your app" />,
  data: <Tip name="Volume" text="an extra disk for a server" />,
  aws: <Tip name="AWS" text="in your own account" />,
  openstack: <Tip name="OpenStack" text="VEXXHOST and other OpenStack clouds" />,
};

// z-10: the layer sits over the hero's text block, so a tile's tip is never
// drawn under the kicker on narrow screens. The tiles keep clear of the text,
// and the layer takes no clicks.
export function HeroCanvas() {
  return (
    <div aria-hidden="true" className="pointer-events-none absolute inset-0 z-10">
      {/* Wide screens: two small diagrams, one per cloud. */}
      <div className="hidden xl:block">
        <div className="absolute" style={clusterBox("left")}>
          <Connector from={LEFT.api} to={LEFT.jobs} kind="http_api" depth={8.5} drawDelayMs={300} />
          <Connector from={LEFT.jobs} to={LEFT.aws} kind="function" depth={13.5} drawDelayMs={700} packetDelay={LEFT_PACKET_DELAY_S} />
          <Tile box={LEFT.api} depth={6} float={{ y: 3.4, x: 4.6, phase: 1.1 }} tilt="-rotate-3" tip={TIPS.api} tipSide="left">
            <BlockFace icon={Globe} kind="http_api" name="api" label="HTTP API" />
          </Tile>
          <Tile box={LEFT.jobs} depth={11} float={{ y: 4.2, x: 3.6, phase: 2.9 }} tilt="rotate-2" tip={TIPS.jobs} tipSide="left">
            <BlockFace icon={Zap} kind="function" name="jobs" label="Function" />
          </Tile>
          <Tile box={LEFT.aws} depth={16} float={{ y: 4.8, x: 4.1, phase: 0.4 }} tilt="-rotate-6" tip={TIPS.aws} tipSide="left">
            <LogoFace connector="aws" size={40} arriveDelay={LEFT_PACKET_DELAY_S} />
          </Tile>
        </div>

        <div className="absolute" style={clusterBox("right")}>
          <Connector from={RIGHT.data} to={RIGHT.web} kind="volume" depth={7} drawDelayMs={450} />
          <Connector from={RIGHT.web} to={RIGHT.openstack} kind="vm" depth={11.5} drawDelayMs={850} packetDelay={RIGHT_PACKET_DELAY_S} />
          <Tile box={RIGHT.openstack} depth={14} float={{ y: 3.7, x: 4.9, phase: 3.3 }} tilt="rotate-6" tip={TIPS.openstack} tipSide="right">
            <LogoFace connector="general-openstack" size={40} arriveDelay={RIGHT_PACKET_DELAY_S} />
          </Tile>
          <Tile box={RIGHT.web} depth={9} float={{ y: 4.5, x: 3.3, phase: 1.8 }} tilt="rotate-2" tip={TIPS.web} tipSide="right">
            <BlockFace icon={Server} kind="vm" name="web" label="VM" />
          </Tile>
          <Tile box={RIGHT.data} depth={5} float={{ y: 3.1, x: 4.4, phase: 2.2 }} tilt="-rotate-2" tip={TIPS.data} tipSide="right">
            <BlockFace icon={HardDrive} kind="volume" name="data" label="Volume" />
          </Tile>
        </div>
      </div>

      {/* Narrower screens: three tiles in the band above the kicker, no lines. */}
      <div className="xl:hidden">
        <Tile className="top-20 left-4 h-[50px] w-28 md:top-24 md:left-[6%]" float={{ y: 3.6, x: 4.4, phase: 1.2 }} tilt="-rotate-3" tip={TIPS.web} tipSide="left">
          <BlockFace icon={Server} kind="vm" name="web" label="VM" />
        </Tile>
        <Tile className="top-[74px] right-4 size-[46px] md:top-[88px] md:right-[6%]" float={{ y: 4.4, x: 3.8, phase: 0.6 }} tilt="rotate-6" tip={TIPS.aws} tipSide="right">
          <LogoFace connector="aws" size={28} />
        </Tile>
        <Tile
          className="top-[82px] right-[70px] size-[46px] md:top-[108px] md:right-[calc(6%+64px)]"
          float={{ y: 3.3, x: 4.8, phase: 2.4 }}
          tilt="-rotate-6"
          tip={TIPS.openstack}
          tipSide="right">
          <LogoFace connector="general-openstack" size={28} />
        </Tile>
      </div>
    </div>
  );
}
