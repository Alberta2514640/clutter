import Link from "next/link";
import motion from "./motion.module.css";
import { FOCUS_RING } from "./site";

const link = `rounded text-teal-300 underline underline-offset-4 hover:text-teal-200 ${FOCUS_RING}`;

const faqs: { q: string; a: React.ReactNode }[] = [
  {
    q: "Which clouds can I deploy to?",
    a: (
      <p>
        Amazon Web Services, and any OpenStack cloud, public or private. Patrii Cloud, a Canadian cloud, is planned once it opens its API. One diagram can have a deployment on each cloud you link.
      </p>
    ),
  },
  {
    q: "What does it cost to try?",
    a: (
      <p>
        Clutter has no paid plans today. Signing in, drawing and previewing cost nothing. Your cloud provider bills you for what a deployment builds in your account, at its usual prices, so destroy a test
        deployment when you are done with it.
      </p>
    ),
  },
  {
    q: "Where does my data live?",
    a: (
      <p>
        Clutter runs on AWS in Canada (Central), and its database is in Canada too. The resources you deploy live in your own cloud account, in the region you choose. The{" "}
        <Link href="/privacy" className={link}>
          privacy policy
        </Link>{" "}
        lists what Clutter keeps and for how long.
      </p>
    ),
  },
  {
    q: "Can I see the Terraform?",
    a: (
      <p>
        Yes. Preview Terraform shows the files Clutter writes for each deployment, before anything runs. Plan then shows what those files would change in your cloud. You do not need to know Terraform to use
        Clutter.
      </p>
    ),
  },
  {
    q: "What if a cloud cannot build one of my blocks?",
    a: (
      <p>
        Clutter tells you before you deploy. The block is marked on that cloud&apos;s view, and the deployment says what it cannot build. You can leave the block out of that one deployment and keep it in the
        others.
      </p>
    ),
  },
  {
    q: "Can I remove Clutter's access to my cloud?",
    a: (
      <p>
        Yes, at any time. Destroy your deployments first if you want their resources removed. Then delete the connection in Clutter. Finally, delete the CloudFormation stack in your AWS account, or the
        application credential in your OpenStack cloud. After that, Clutter can no longer reach your account.
      </p>
    ),
  },
  {
    q: "Who can see my diagrams and connections?",
    a: <p>Only members of your organization. Each organization has its own projects, diagrams and connections.</p>,
  },
];

// Native disclosure widgets: keyboard and screen-reader support with no script.
export function Faq() {
  return (
    <section id="faq" aria-labelledby="faq-title" className="scroll-mt-16 border-t border-slate-800/60 bg-gradient-to-b from-black via-slate-950 to-black">
      <div className="mx-auto max-w-3xl px-4 py-16 md:px-6 md:py-24">
        <p className="mb-3 text-sm font-semibold uppercase tracking-[0.2em] text-cyan-300">FAQ</p>
        <h2 id="faq-title" className="mb-8 text-3xl font-semibold tracking-tight text-white md:text-4xl">
          Questions people ask
        </h2>
        <div className={`divide-y divide-slate-800 rounded-2xl border border-slate-800 bg-slate-950/70 ${motion.reveal}`}>
          {faqs.map(({ q, a }) => (
            <details key={q} className="group px-5 py-1">
              <summary
                className={`flex cursor-pointer list-none items-center justify-between gap-4 rounded py-4 font-medium text-white marker:content-none [&::-webkit-details-marker]:hidden ${FOCUS_RING}`}>
                <h3>{q}</h3>
                <span aria-hidden="true" className="text-xl leading-none text-teal-300 transition-transform group-open:rotate-45 motion-reduce:transition-none">
                  +
                </span>
              </summary>
              <div className="pb-5 leading-relaxed text-slate-300">{a}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}
