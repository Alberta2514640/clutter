import { Cloud } from "lucide-react";
import Link from "next/link";

// The path from a new organization to a first deploy, owned once for the
// onboarding page and the empty dashboard. withLinks is false before the
// organization exists, when the pages it would link to have nothing to show.
// The first step is the one to do now, so it is a button, not a text link.
const steps = [
  {
    title: "Connect a cloud",
    detail: "Link the cloud account Clutter deploys into, such as AWS or OpenStack.",
    href: "/settings/organization/connections",
    linkLabel: "Connect a cloud",
    button: true,
  },
  // No link here: the page around the steps has the one New project button.
  {
    title: "Create a project",
    detail: "A project holds your diagrams.",
  },
  {
    title: "Draw a diagram",
    detail: "Drag building blocks such as servers, networks and storage onto the canvas.",
  },
  {
    title: "Deploy",
    detail: "Preview the Terraform, then Plan and Deploy to each connected cloud. Destroy removes it again.",
  },
];

export default function FirstSteps({ withLinks = true }: { withLinks?: boolean }) {
  return (
    <ol className="space-y-3 text-sm">
      {steps.map((step, index) => (
        <li key={step.title} className="flex gap-3">
          <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-teal-500/60 bg-teal-500/20 text-xs text-white" aria-hidden="true">
            {index + 1}
          </span>
          <div>
            <p className="font-medium text-white">{step.title}</p>
            <p className="text-gray-400">
              {step.detail}
              {withLinks && step.href && !step.button && (
                <>
                  {" "}
                  <Link href={step.href} className="text-teal-300 underline underline-offset-2 hover:text-teal-200">
                    {step.linkLabel}
                  </Link>
                </>
              )}
            </p>
            {withLinks && step.href && step.button && (
              <Link
                href={step.href}
                className="mt-2 inline-flex items-center gap-1.5 rounded-md bg-teal-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-teal-500 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-teal-300">
                <Cloud className="h-4 w-4" aria-hidden="true" />
                {step.linkLabel}
              </Link>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}
