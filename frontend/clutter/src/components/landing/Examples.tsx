import SpotlightGroup from "@/components/common/SpotlightGroup";
import LevelBadge from "@/components/common/LevelBadge";
import { ArrowRight } from "lucide-react";
import Link from "next/link";
import { ExampleClouds } from "./ExampleClouds";
import { LANDING_EXAMPLES } from "./exampleList";
import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";
import { FOCUS_RING } from "./site";

// The built-in examples (exampleList.ts), as cards. No count is stated.
export function Examples() {
  return (
    <section id="examples" aria-labelledby="examples-title" className="border-border bg-card grain scroll-mt-20 border-y py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4 md:px-8">
        <div className={`flex flex-col gap-6 md:flex-row md:items-end md:justify-between ${motion.reveal}`}>
          <div className="max-w-2xl">
            <Kicker>Examples</Kicker>
            <SectionHeading id="examples-title">Start from an example.</SectionHeading>
            <p className="text-muted-foreground mt-5 text-lg leading-relaxed">
              Each one is a working diagram, from one server to an app split across two clouds. Open it, change what you like, and deploy.
            </p>
          </div>
          <Link href="/login" className={`text-link inline-flex shrink-0 items-center gap-1.5 rounded font-medium hover:underline hover:underline-offset-4 ${FOCUS_RING}`}>
            Sign in to open one
            <ArrowRight className="size-4" aria-hidden="true" />
          </Link>
        </div>

        <SpotlightGroup as="ul" className="mt-12 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {LANDING_EXAMPLES.map((e, i) => (
            <li key={e.id} style={{ "--reveal-delay": `${(i % 3) * 60}ms` } as React.CSSProperties} className={motion.reveal}>
              <div data-spotlight className="surface-card spotlight relative flex h-full flex-col gap-4 p-5">
                <div className="flex items-start justify-between gap-3">
                  <h3 className="text-lg font-semibold">{e.name}</h3>
                  <LevelBadge level={e.level} />
                </div>
                <p className="text-muted-foreground flex-1 leading-relaxed">{e.description}</p>
                <ExampleClouds clouds={e.clouds} split={e.split} />
              </div>
            </li>
          ))}
        </SpotlightGroup>
      </div>
    </section>
  );
}
