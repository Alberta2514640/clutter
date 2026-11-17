import type { Metadata } from "next";
import { ContactEmail } from "@/components/landing/ContactEmail";
import { LegalPage } from "@/components/landing/LegalPage";
import { OPERATOR } from "@/components/landing/site";

export const metadata: Metadata = {
  title: "Privacy policy",
  description: "What Clutter collects, why, where it is stored, how long it is kept, and how to use your privacy rights.",
  alternates: { canonical: "/privacy" },
  robots: { index: true, follow: true },
};

export default function PrivacyPage() {
  return (
    <LegalPage
      title="Privacy policy"
      intro={
        <p>
          Clutter is run by {OPERATOR} (&quot;we&quot;). This page says what we collect when you use Clutter, why, where it is stored, how long we keep it, and what you can ask us to do with it.
        </p>
      }>
      <section aria-labelledby="collect">
        <h2 id="collect">What we collect</h2>
        <p>
          <strong>When you sign in with Google:</strong> your name, your email address and a link to your Google profile picture. Google confirms who you are. We never see your Google password.
        </p>
        <p>
          <strong>What you create in Clutter:</strong> your organizations, projects and diagrams; your connections and deployments and their settings; the Terraform files Clutter writes; the logs of each
          run; and any function code you upload.
        </p>
        <p>
          <strong>What a connection needs:</strong> for AWS, the ARN of the role you created for Clutter and its external id. For OpenStack, your cloud&apos;s address and an application credential (its ID
          and secret). We store these encrypted. Clutter never shows a secret again after you save it.
        </p>
        <p>We do not collect payment details. Clutter has no paid plans today.</p>
      </section>

      <section aria-labelledby="why">
        <h2 id="why">Why we use it</h2>
        <p>We use this information only to run Clutter for you:</p>
        <ul>
          <li>to sign you in and keep you signed in;</li>
          <li>to save your work and show it to the members of your organization;</li>
          <li>to write Terraform from your diagrams and to plan, deploy and destroy it in the cloud accounts you link, when you ask;</li>
          <li>to keep Clutter secure and to find and fix faults.</li>
        </ul>
      </section>

      <section aria-labelledby="where">
        <h2 id="where">Where it is stored, and who helps us</h2>
        <p>Clutter runs in Canada. These companies process your information for us:</p>
        <ul>
          <li>
            <strong>Amazon Web Services</strong>, in its Canada (Central) region (ca-central-1): Clutter&apos;s servers, the files Clutter writes, run logs, uploaded function code, and your encrypted
            connection credentials.
          </li>
          <li>
            <strong>Supabase</strong>, in Canada (Central): Clutter&apos;s database, which holds your account, organizations, projects, diagrams, connections and deployments.
          </li>
          <li>
            <strong>Google</strong>: sign-in. Google&apos;s own privacy policy applies to your Google account.
          </li>
        </ul>
        <p>
          When you deploy, Clutter calls your cloud provider with the access you gave it. The resources it builds live in your own cloud account, and your agreement with that provider applies to them.
        </p>
      </section>

      <section aria-labelledby="never">
        <h2 id="never">What we never do</h2>
        <ul>
          <li>We do not sell your information or share it for advertising.</li>
          <li>We show no ads, and we use no analytics or tracking tools.</li>
          <li>
            Clutter keeps only these things in your browser: a sign-in cookie and a copy of your sign-in token, and a few settings such as the organization you chose and how you last viewed a diagram.
            Google&apos;s sign-in button may keep its own cookies, under Google&apos;s policy.
          </li>
        </ul>
      </section>

      <section aria-labelledby="how-long">
        <h2 id="how-long">How long we keep it</h2>
        <ul>
          <li>
            <strong>Run logs:</strong> 90 days.
          </li>
          <li>
            <strong>Terraform state and the files Clutter writes:</strong> while the deployment exists. Older copies of a deployment&apos;s state are deleted after 90 days.
          </li>
          <li>
            <strong>Uploaded function code:</strong> until you replace or remove it.
          </li>
          <li>
            <strong>Your account and what you created:</strong> until you delete it, or ask us to delete it.
          </li>
          <li>
            <strong>Logs of Clutter&apos;s own services:</strong> these record requests and errors, never your credentials. Most are deleted after 30 days or less. Some older parts of Clutter keep their logs
            until we delete them.
          </li>
        </ul>
      </section>

      <section aria-labelledby="rights">
        <h2 id="rights">Your rights</h2>
        <p>Canada&apos;s privacy law (PIPEDA) gives you the right to:</p>
        <ul>
          <li>see the personal information we hold about you;</li>
          <li>have it corrected;</li>
          <li>have it deleted;</li>
          <li>withdraw your consent. Clutter cannot run without sign-in, so withdrawing consent means closing your account.</li>
        </ul>
        <p>
          To use any of these rights, write to <ContactEmail />. We answer within 30 days. If you are not happy with our answer, you can complain to the Office of the Privacy Commissioner of Canada.
        </p>
        <p>
          <strong>To remove Clutter&apos;s access to a cloud account:</strong> destroy the deployments you want removed, delete the connection in Clutter, then delete the CloudFormation stack in your AWS
          account, or the application credential in your OpenStack cloud.
        </p>
      </section>

      <section aria-labelledby="security">
        <h2 id="security">How we protect it</h2>
        <ul>
          <li>Connection credentials are encrypted when stored, and are never shown again after you save them.</li>
          <li>Only members of your organization can see or use its projects, diagrams, connections and deployments.</li>
          <li>Your browser talks to Clutter over HTTPS.</li>
        </ul>
        <p>No system is perfectly secure. If we learn of a breach that puts you at real risk, we will tell you and the Privacy Commissioner, as the law requires.</p>
      </section>

      <section aria-labelledby="changes">
        <h2 id="changes">Changes to this policy</h2>
        <p>When we change this policy, we update this page and the date at the top.</p>
      </section>

      <section aria-labelledby="contact">
        <h2 id="contact">Contact</h2>
        <p>
          Questions about your privacy go to <ContactEmail />.
        </p>
      </section>
    </LegalPage>
  );
}
