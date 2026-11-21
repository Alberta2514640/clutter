import type { Metadata } from "next";
import Link from "next/link";
import { ContactEmail } from "@/components/landing/ContactEmail";
import { LegalPage } from "@/components/landing/LegalPage";
import { FOCUS_RING, OPERATOR } from "@/components/landing/site";

export const metadata: Metadata = {
  title: "Terms of use",
  description: "The terms for using Clutter: what the service is, your responsibilities for your cloud accounts, acceptable use, and the law that applies.",
  alternates: { canonical: "/terms" },
  robots: { index: true, follow: true },
};

export default function TermsPage() {
  return (
    <LegalPage
      title="Terms of use"
      intro={
        <p>
          These terms are an agreement between you and {OPERATOR} (&quot;we&quot;), which runs Clutter. By signing in to Clutter, you agree to them. If you use Clutter for a company, you agree for that
          company too.
        </p>
      }>
      <section aria-labelledby="service">
        <h2 id="service">What Clutter is</h2>
        <p>
          Clutter lets you draw cloud infrastructure as a diagram of building blocks. It turns the diagram into Terraform and, when you ask, runs that Terraform in the cloud accounts you link to it. It
          can also destroy what it built.
        </p>
      </section>

      <section aria-labelledby="early">
        <h2 id="early">An early-stage service</h2>
        <p>
          Clutter is at an early stage. We provide it &quot;as is&quot; and &quot;as available&quot;, without any warranty, express or implied, including any warranty that it is fit for a particular
          purpose, error-free or always available. Features may change or stop. Keep your own copy of anything you cannot afford to lose.
        </p>
      </section>

      <section aria-labelledby="account">
        <h2 id="account">Your Clutter account</h2>
        <p>
          You sign in with a Google account. Keep that account secure. You are responsible for what happens in your organizations, including what other members do there.
        </p>
      </section>

      <section aria-labelledby="clouds">
        <h2 id="clouds">Your cloud accounts</h2>
        <p>You stay responsible for the cloud accounts you link. In particular, you are responsible for:</p>
        <ul>
          <li>having the right to give Clutter access to each account;</li>
          <li>the resources Clutter builds in your accounts at your request, and what runs on them;</li>
          <li>every charge from your cloud provider, including charges for resources you forget to destroy;</li>
          <li>following your cloud provider&apos;s own terms.</li>
        </ul>
        <p>Read the Terraform preview and the plan before you deploy. Destroy a deployment when you no longer need what it built.</p>
      </section>

      <section aria-labelledby="content">
        <h2 id="content">Your content</h2>
        <p>
          You own your diagrams, settings and uploaded code. You let us store and process them only to run Clutter for you. The{" "}
          <Link href="/privacy" className={`text-link rounded underline underline-offset-4 ${FOCUS_RING}`}>
            privacy policy
          </Link>{" "}
          says how we handle your information.
        </p>
      </section>

      <section aria-labelledby="use">
        <h2 id="use">Acceptable use</h2>
        <p>Do not use Clutter to:</p>
        <ul>
          <li>break any law, or build infrastructure for an unlawful purpose;</li>
          <li>attack, probe, overload or get around the security of Clutter;</li>
          <li>reach another person&apos;s account, organization, data or cloud account without their permission;</li>
          <li>upload code meant to cause harm;</li>
          <li>abuse a cloud provider or its other customers.</li>
        </ul>
      </section>

      <section aria-labelledby="end">
        <h2 id="end">Suspension and termination</h2>
        <p>
          We may suspend or close your access if you break these terms, or if we must to protect Clutter, its users or a cloud provider. When we can, we will tell you first. You can stop using Clutter at
          any time and ask us to delete your account.
        </p>
        <p>Closing an account does not remove resources in your cloud accounts. Destroy your deployments first if you want them removed.</p>
      </section>

      <section aria-labelledby="liability">
        <h2 id="liability">Limitation of liability</h2>
        <p>
          As far as the law allows, we are not liable for indirect, incidental or consequential losses, for lost data or profits, or for charges from your cloud provider. As far as the law allows, our
          total liability for any claim about Clutter is limited to the greater of the amount you paid us for Clutter in the 12 months before the claim and 100 Canadian dollars.
        </p>
      </section>

      <section aria-labelledby="changes">
        <h2 id="changes">Changes to these terms</h2>
        <p>When we change these terms, we update this page and the date at the top. If you keep using Clutter after a change, the new terms apply.</p>
      </section>

      <section aria-labelledby="law">
        <h2 id="law">Governing law</h2>
        <p>These terms are governed by the laws of Alberta and the federal laws of Canada that apply there. The courts of Alberta decide any dispute about them.</p>
      </section>

      <section aria-labelledby="contact">
        <h2 id="contact">Contact</h2>
        <p>
          Questions about these terms go to <ContactEmail />.
        </p>
      </section>
    </LegalPage>
  );
}
