import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";
import { ViewAsDemo } from "./ViewAsDemo";

// The "View as" switch and its compatibility check, tried on the page.
export function SeeEachCloud() {
  return (
    <section id="view-as" aria-labelledby="view-title" className="border-border bg-card grain scroll-mt-20 border-y py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4 md:px-8">
        <div className={`max-w-3xl ${motion.reveal}`}>
          <Kicker>Check</Kicker>
          <SectionHeading id="view-title">See how each cloud builds it.</SectionHeading>
          <p className="text-muted-foreground mt-5 text-lg leading-relaxed">
            Switch the view to a cloud, and each block shows the service that cloud uses. A block the cloud cannot build is marked, so you know before you deploy. Try it here.
          </p>
        </div>
        <div className={`mt-10 ${motion.reveal}`}>
          <ViewAsDemo sizes="(min-width: 1216px) 1152px, calc(100vw - 32px)" />
        </div>
      </div>
    </section>
  );
}
