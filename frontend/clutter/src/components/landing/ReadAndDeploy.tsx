import motion from "./motion.module.css";
import { Screenshot } from "./Screenshot";
import { Kicker, SectionHeading } from "./Section";
import { SHOTS } from "./shots";

const steps = [
  { title: "Preview the Terraform", text: "Read the files Clutter writes for this deployment, before anything runs. You do not need to know Terraform to use Clutter." },
  { title: "Plan", text: "See what those files would change in your cloud." },
  { title: "Confirm", text: "Clutter says what it will build and names the costly part, such as a NAT gateway on AWS." },
  { title: "Deploy, and destroy when done", text: "Follow the run log as it happens. Destroy removes what the deployment built." },
];

// Read the code, then deploy: the steps and the Terraform preview beside the
// deploy panel's shot.
export function ReadAndDeploy() {
  return (
    <section id="deploy" aria-labelledby="deploy-title" className="scroll-mt-20 py-20 md:py-28">
      <div className="mx-auto grid max-w-6xl items-center gap-12 px-4 md:px-8 lg:grid-cols-[7fr_5fr] lg:gap-16">
        <div className={motion.reveal}>
          <Kicker>Deploy</Kicker>
          <SectionHeading id="deploy-title">Read the code, then deploy.</SectionHeading>
          <ol className="mt-10 space-y-6">
            {steps.map((s, i) => (
              <li key={s.title} className="flex gap-4">
                <span aria-hidden="true" className="border-primary/40 text-link flex size-8 shrink-0 items-center justify-center rounded-full border text-sm font-semibold">
                  {i + 1}
                </span>
                <div>
                  <h3 className="font-semibold">{s.title}</h3>
                  <p className="text-muted-foreground mt-1 leading-relaxed">{s.text}</p>
                </div>
              </li>
            ))}
          </ol>
          <Screenshot
            shot={SHOTS.preview}
            sizes="(min-width: 1216px) 620px, (min-width: 1024px) 55vw, calc(100vw - 32px)"
            caption="The Terraform preview: every file Clutter writes for a deployment, before anything runs."
            className="mt-10"
          />
        </div>
        <Screenshot shot={SHOTS.deploy} sizes="(min-width: 1024px) 420px, min(440px, calc(100vw - 32px))" className={`mx-auto w-full max-w-[440px] ${motion.reveal}`} />
      </div>
    </section>
  );
}
