import { Plus } from "lucide-react";
import Link from "next/link";
import { ContactEmail } from "./ContactEmail";
import motion from "./motion.module.css";
import { Kicker, SectionHeading } from "./Section";
import { CONTACT_EMAIL, FOCUS_RING } from "./site";

const link = `text-link rounded underline underline-offset-4 ${FOCUS_RING}`;

const faqs: { q: string; a: React.ReactNode }[] = [
  {
    q: "Which clouds can I deploy to?",
    a: (
      <p>
        Amazon Web Services, and OpenStack clouds such as VEXXHOST. Patrii Cloud, a Canadian cloud, is planned once it opens its API. One diagram can have a deployment on
        each cloud you link.
      </p>
    ),
  },
  {
    q: "What does it cost to try?",
    a: (
      <p>
        Signing in, drawing and previewing are free. Your cloud provider bills you for what a deployment builds in your account, at its usual prices, so destroy a test
        deployment when you are done with it.
      </p>
    ),
  },
  {
    q: "Where does my data live?",
    a: (
      <p>
        What you deploy lives in your own cloud account, in the region you choose. Clutter keeps your diagrams and connection details, and the{" "}
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
        Yes. Preview code shows the Terraform files Clutter writes for each deployment, before anything runs. Plan then shows what those files would change in your cloud. You do not need to know Terraform to use
        Clutter.
      </p>
    ),
  },
  {
    q: "What if a cloud cannot build one of my blocks?",
    a: (
      <p>
        Clutter tells you before you deploy. The block is marked in that cloud&apos;s view. You can run it on another cloud you linked, in the same deployment, or leave it out of that one deployment and keep
        it in the others.
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
    a: (
      <p>
        Only the people in your team. Each team has its own projects, diagrams and connections.
      </p>
    ),
  },
];

// Two columns: the heading on the left, native disclosure widgets on the
// right (keyboard and screen-reader support with no script).
export function Faq() {
  return (
    <section id="faq" aria-labelledby="faq-title" className="border-border scroll-mt-20 border-t py-20 md:py-28">
      <div className="mx-auto grid max-w-6xl gap-10 px-4 md:px-8 lg:grid-cols-[4fr_7fr] lg:gap-16">
        <div className={motion.reveal}>
          <Kicker>FAQ</Kicker>
          <SectionHeading id="faq-title">Questions people ask.</SectionHeading>
          {CONTACT_EMAIL && (
            <p className="text-muted-foreground mt-5 leading-relaxed">
              Something else? Write to <ContactEmail />.
            </p>
          )}
        </div>
        <div className={`surface-card divide-border divide-y ${motion.reveal}`}>
          {faqs.map(({ q, a }) => (
            <details key={q} className="group px-5">
              <summary className={`flex min-h-14 cursor-pointer list-none items-center justify-between gap-4 rounded py-4 font-medium marker:content-none [&::-webkit-details-marker]:hidden ${FOCUS_RING}`}>
                <h3>{q}</h3>
                <Plus aria-hidden="true" className="text-link motion-base size-5 shrink-0 transition-transform group-open:rotate-45" />
              </summary>
              <div className="text-muted-foreground pb-5 leading-relaxed">{a}</div>
            </details>
          ))}
        </div>
      </div>
    </section>
  );
}
