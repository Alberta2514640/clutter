import CloudLogo from "@/components/common/CloudLogo";
import { Badge } from "@/components/ui/badge";
import motion from "./motion.module.css";
import { Screenshot } from "./Screenshot";
import { Kicker, SectionHeading } from "./Section";
import { SHOTS } from "./shots";

// "Available" means a real account built and removed every block listed
// (docs/reports/2026-11-20-real-cloud-verification.pdf); the page states what
// each cloud builds, not the test itself.
// Patrii is named in text only, with no logo (ADR-022).
const clouds = [
  {
    connector: "aws",
    name: "Amazon Web Services",
    available: true,
    lines: [
      "Builds every block: VMs, networks, firewalls, volumes, buckets, functions, tables and HTTP APIs.",
      "Link an account with a CloudFormation stack that makes a role for Clutter.",
    ],
  },
  {
    connector: "general-openstack",
    name: "OpenStack",
    available: true,
    lines: [
      "Builds VMs, networks, firewalls and volumes. The other blocks can run on AWS in the same deployment.",
      "Connect with an application credential, or drop in the clouds.yaml file your cloud gives you.",
    ],
  },
  {
    connector: null,
    name: "Patrii Cloud",
    available: false,
    lines: ["A Canadian cloud. Planned once Patrii opens its API to customers."],
  },
];

export function Clouds() {
  return (
    <section id="clouds" aria-labelledby="clouds-title" className="scroll-mt-20 py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4 md:px-8">
        <div className={`max-w-3xl ${motion.reveal}`}>
          <Kicker>Clouds</Kicker>
          <SectionHeading id="clouds-title">Clouds you can link.</SectionHeading>
          <p className="text-muted-foreground mt-5 text-lg leading-relaxed">
            Clutter checks each connection before it saves it, and every project in your team can then use it.
          </p>
        </div>

        <ul className="mt-12 grid gap-4 md:grid-cols-3">
          {clouds.map((c, i) => (
            <li key={c.name} style={{ "--reveal-delay": `${i * 60}ms` } as React.CSSProperties} className={motion.reveal}>
              <div className="surface-card flex h-full flex-col p-6">
                <div className="mb-5 flex items-center justify-between gap-3">
                  <h3 className="flex items-center gap-3 text-lg font-semibold">
                    {c.connector && <CloudLogo connector={c.connector} size={28} />}
                    {c.name}
                  </h3>
                  <Badge variant={c.available ? "success" : "neutral"}>{c.available ? "Available" : "Planned"}</Badge>
                </div>
                <ul className="text-muted-foreground flex-1 space-y-2 text-sm leading-relaxed">
                  {c.lines.map((l) => (
                    <li key={l}>{l}</li>
                  ))}
                </ul>
              </div>
            </li>
          ))}
        </ul>

        <Screenshot
          shot={SHOTS.connections}
          sizes="(min-width: 1216px) 1152px, calc(100vw - 32px)"
          caption="The Connections page: each connection is checked, then any project in the team can deploy to it."
          className={`mt-12 ${motion.reveal}`}
        />
        <p className="text-muted-foreground mt-6 text-sm">Cloud names and logos belong to their owners. Clutter is not made or endorsed by them.</p>
      </div>
    </section>
  );
}
