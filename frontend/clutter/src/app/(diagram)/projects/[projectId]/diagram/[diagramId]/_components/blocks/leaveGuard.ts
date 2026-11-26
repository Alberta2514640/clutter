// The click half of the editor's leave guard (useLeaveGuard), without React:
// a click listener in the capture phase on the document runs before React's
// handlers and Next's Link, and holds a click on a link inside the app. When
// the person chooses to leave, the held link is clicked again and let
// through, so its own behaviour runs (a breadcrumb or Connections link
// chooses its team first) and Next navigates as it would have.

// A held link: where it goes, and the link itself to click again.
export type HeldLink = { href: string; link: HTMLAnchorElement | null };

// heldLink is the link a click would follow inside the app, or null for a
// click the browser keeps: a modified or non-left click, one already handled,
// a link to another site, to a new tab or a download, or to this same page.
function heldLink(e: MouseEvent, here: { origin: string; pathname: string; href: string }): HeldLink | null {
  if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return null;
  const link = e.target instanceof Element ? e.target.closest("a[href]") : null;
  if (!link || link.tagName !== "A") return null;
  const a = link as HTMLAnchorElement;
  if ((a.target && a.target !== "_self") || a.hasAttribute("download")) return null;
  const url = new URL(a.href, here.href);
  if (url.origin !== here.origin) return null;
  if (url.pathname === here.pathname) return null;
  return { href: url.pathname + url.search + url.hash, link: a };
}

// holdLinkClicks starts holding link clicks on doc, handing each held link to
// onHold. follow clicks a held link again and lets that click through; it
// answers false when the link is gone from the page, and the caller then goes
// to its href.
export function holdLinkClicks(doc: Document, onHold: (held: HeldLink) => void) {
  let letThrough = false;
  const view = doc.defaultView;
  const onClick = (e: MouseEvent) => {
    if (letThrough) {
      letThrough = false;
      return;
    }
    if (!view) return;
    const held = heldLink(e, view.location);
    if (!held) return;
    e.preventDefault();
    e.stopPropagation();
    onHold(held);
  };
  doc.addEventListener("click", onClick, true);
  return {
    follow(held: HeldLink): boolean {
      if (!held.link?.isConnected) return false;
      letThrough = true;
      try {
        held.link.click();
      } finally {
        letThrough = false;
      }
      return true;
    },
    dispose: () => doc.removeEventListener("click", onClick, true),
  };
}
