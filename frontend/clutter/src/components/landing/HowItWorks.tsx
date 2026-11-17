import motion from "./motion.module.css";
import { Screenshot } from "./Screenshot";
import { SHOTS } from "./shots";
import { ViewAsDemo } from "./ViewAsDemo";

// One step. From lg up its number sits on the rail that joins the three steps.
function Step({ n, title, children }: { n: number; title: string; children: React.ReactNode }) {
  return (
    <div className="lg:relative">
      <p className="mb-3 flex h-8 w-8 items-center justify-center rounded-full border border-teal-400/60 bg-[#04101a] text-sm font-semibold text-teal-200 shadow-[0_0_0_6px_#000] lg:absolute lg:-left-16 lg:top-0 lg:mb-0">
        <span className="sr-only">Step </span>
        {n}
      </p>
      <h3 className="mb-3 text-2xl font-semibold tracking-tight text-white">{title}</h3>
      <div className="space-y-3 leading-relaxed text-slate-300">{children}</div>
    </div>
  );
}

export function HowItWorks() {
  return (
    <section id="how-it-works" aria-labelledby="how-title" className="scroll-mt-16 bg-black">
      <div className="mx-auto max-w-6xl px-4 py-16 md:px-6 md:py-24">
        <div className={`mb-12 max-w-2xl ${motion.reveal}`}>
          <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">How it works</p>
          <h2 id="how-title" className="text-3xl font-semibold tracking-tight text-white md:text-4xl">
            From a drawing to running infrastructure, in three steps.
          </h2>
          <p className="mt-4 leading-relaxed text-slate-300">Nothing is built until you press Deploy. Every step happens in your browser.</p>
        </div>

        <ol className="relative space-y-20 lg:before:absolute lg:before:bottom-8 lg:before:left-4 lg:before:top-4 lg:before:w-px lg:before:bg-gradient-to-b lg:before:from-teal-400/70 lg:before:via-indigo-400/40 lg:before:to-teal-400/10">

          <li className={`relative grid items-center gap-8 lg:grid-cols-[2fr_3fr] lg:pl-16 ${motion.reveal}`}>
            <Step n={1} title="Connect a cloud">
              <p>Link an AWS account with a role you create in it, or add an OpenStack cloud with an application credential.</p>
              <p>Clutter checks each connection before it saves it. Every project in your organization can then use it.</p>
            </Step>
            <Screenshot shot={SHOTS.connections} sizes="(min-width: 1152px) 640px, (min-width: 1024px) 55vw, calc(100vw - 32px)" />
          </li>

          <li className={`relative space-y-8 lg:pl-16 ${motion.reveal}`}>
            <div className="max-w-2xl">
              <Step n={2} title="Draw it once, in building blocks">
                <p>
                  Place VMs, networks, firewalls, volumes, buckets and functions on the canvas, then connect them. Each block has a short form with plain choices, such as a small, medium or large VM.
                </p>
                <p>Switch the view to see how each cloud builds your diagram. A block that a cloud does not offer is marked, so you know before you deploy. Try it here:</p>
              </Step>
            </div>
            <div className="max-w-4xl">
              <ViewAsDemo sizes="(min-width: 1024px) 896px, calc(100vw - 32px)" />
            </div>
          </li>

          <li className={`relative grid items-center gap-8 lg:grid-cols-[3fr_2fr] lg:pl-16 ${motion.reveal}`}>
            <Step n={3} title="Preview, plan and deploy">
              <p>Each linked cloud gets its own deployment of the same diagram, with its own settings. One deployment can also place some blocks on another cloud.</p>
              <p>Read the Terraform that Clutter writes. Run Plan to see what would change. Then press Deploy and follow the run log as it happens.</p>
              <p>Destroy removes everything that deployment built.</p>
            </Step>
            <Screenshot shot={SHOTS.deploy} sizes="(min-width: 1024px) 400px, min(440px, calc(100vw - 32px))" className="mx-auto w-full max-w-[440px]" />
          </li>
        </ol>
      </div>
    </section>
  );
}
