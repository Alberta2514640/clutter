import { CONTACT_EMAIL, FOCUS_RING } from "./site";

// The contact address as a mail link, or a plain note when this build has none.
export function ContactEmail({ className = "" }: { className?: string }) {
  if (!CONTACT_EMAIL) {
    return <span className={className}>(no contact address is set for this site yet)</span>;
  }
  return (
    <a href={`mailto:${CONTACT_EMAIL}`} className={`rounded text-teal-300 underline underline-offset-4 hover:text-teal-200 ${FOCUS_RING} ${className}`}>
      {CONTACT_EMAIL}
    </a>
  );
}
