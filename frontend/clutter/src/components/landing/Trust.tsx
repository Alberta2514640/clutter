import { KeyRound, ShieldCheck, Trash2, Users } from "lucide-react";
import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";

// How an AWS account is linked, as the real-cloud test did it
// (docs/reports/2026-11-20-real-cloud-verification.pdf, pages 1 and 11).
const linkSteps = [
  { who: "You", title: "Open AWS from Clutter", text: "CloudFormation opens in your account with Clutter's template filled in." },
  { who: "Your account", title: "The stack makes a role", text: "Only Clutter can switch into it, and only with your own external id." },
  { who: "Clutter", title: "Clutter checks the link", text: "Paste the role's ARN. Clutter switches into the role to prove it works, then saves it." },
  { who: "Every run", title: "Keys for an hour at most", text: "Each plan, deploy and destroy gets keys from the role that last at most one hour." },
];

// The limits, true as written. The role has exceptions on buckets and
// addresses (ADR-027), so never write "changes only what it built"; the
// first fact names only what the role cannot touch.
const facts = [
  {
    icon: ShieldCheck,
    title: "A narrow role",
    text: "The role cannot change or delete servers, functions, tables or APIs that Clutter did not create.",
  },
  {
    icon: KeyRound,
    title: "No AWS keys to hand over",
    text: "You never give Clutter AWS keys. For OpenStack you add an application credential; Clutter stores it encrypted and never shows it again.",
  },
  {
    icon: Users,
    title: "Only your team",
    text: "Each team has its own projects, diagrams and connections, and only its members see them.",
  },
  {
    icon: Trash2,
    title: "Leave any time",
    text: "Destroy removes what a deployment built. Then delete the connection, and the stack or credential in your cloud, and Clutter can no longer reach it.",
  },
];

export function Trust() {
  return (
    <section id="security" aria-labelledby="security-title" className="border-border bg-card grain scroll-mt-20 border-y py-20 md:py-28">
      <div className="mx-auto max-w-6xl px-4 md:px-8">
        <div className={`max-w-3xl ${motion.reveal}`}>
          <Kicker>Security</Kicker>
          <SectionHeading id="security-title">Your account. A narrow role. No AWS keys.</SectionHeading>
          <p className="text-muted-foreground mt-5 text-lg leading-relaxed">Everything a deployment builds lands in your own cloud account. Here is how Clutter gets in, and what it can touch.</p>
        </div>

        <ol aria-label="How an AWS account is linked" className={`mt-12 grid gap-4 md:grid-cols-4 ${motion.reveal}`}>
          {linkSteps.map((s, i) => (
            <li key={s.title} className="surface-card p-5">
              <p className="micro-label mb-3 flex items-center gap-2">
                <span aria-hidden="true" className="bg-primary text-primary-foreground flex size-5 items-center justify-center rounded-full text-[11px] tracking-normal">
                  {i + 1}
                </span>
                {s.who}
              </p>
              <h3 className="font-semibold">{s.title}</h3>
              <p className="text-muted-foreground mt-1.5 text-sm leading-relaxed">{s.text}</p>
            </li>
          ))}
        </ol>

        <ul className={`mt-12 grid gap-x-10 gap-y-8 md:grid-cols-2 ${motion.reveal}`}>
          {facts.map(({ icon: Icon, title, text }) => (
            <li key={title} className="flex gap-4">
              <span className="border-primary/25 bg-primary/10 flex size-10 shrink-0 items-center justify-center rounded-lg border">
                <Icon className="text-link size-5" aria-hidden="true" />
              </span>
              <div>
                <h3 className="font-semibold">{title}</h3>
                <p className="text-muted-foreground mt-1.5 leading-relaxed">{text}</p>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
