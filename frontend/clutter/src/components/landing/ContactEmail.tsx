import { CONTACT_EMAIL, FOCUS_RING } from "./site";

// The contact address as a mail link, or a plain note when this build has none
// (the privacy and terms pages need a sentence either way; the home page shows
// its contact line only when there is an address).
export function ContactEmail({ className = "" }: { className?: string }) {
  if (!CONTACT_EMAIL) {
    return <span className={className}>(no contact address is set for this site yet)</span>;
  }
  return (
    <a href={`mailto:${CONTACT_EMAIL}`} className={`text-link rounded underline underline-offset-4 ${FOCUS_RING} ${className}`}>
      {CONTACT_EMAIL}
    </a>
  );
}
