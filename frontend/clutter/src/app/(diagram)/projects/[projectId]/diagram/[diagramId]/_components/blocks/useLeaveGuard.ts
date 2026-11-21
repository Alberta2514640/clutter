"use client";

import { useRouter } from "next/navigation";
import { useCallback, useEffect, useRef, useState } from "react";
import { holdLinkClicks, type HeldLink } from "./leaveGuard";

// While the diagram has unsaved changes, a link inside the app asks first in
// the app's own dialog (LeaveDialog), and the browser asks before the tab
// closes or reloads. The App Router has no navigation event to cancel, so the
// click is held before Next's Link sees it (leaveGuard.ts). "Leave without
// saving" clicks the link again and lets it through, so the link's own
// onClick runs (a breadcrumb chooses its team first). A link to another
// site, or one opened in a new tab, is left to the browser.
export function useLeaveGuard(dirty: boolean) {
  const router = useRouter();
  // Where the person tried to go, while the dialog asks.
  const [pending, setPending] = useState<HeldLink | null>(null);
  const clicks = useRef<ReturnType<typeof holdLinkClicks> | null>(null);

  useEffect(() => {
    if (!dirty) return;
    const onBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
      e.returnValue = "";
    };
    const held = holdLinkClicks(document, setPending);
    clicks.current = held;
    window.addEventListener("beforeunload", onBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", onBeforeUnload);
      held.dispose();
      if (clicks.current === held) clicks.current = null;
    };
  }, [dirty]);

  // leave goes to href, asking first while there are unsaved changes.
  const leave = useCallback(
    (href: string) => {
      if (dirty) setPending({ href, link: null });
      else router.push(href);
    },
    [dirty, router],
  );

  return {
    pending,
    leave,
    stay: () => setPending(null),
    confirm: () => {
      const held = pending;
      setPending(null);
      if (!held) return;
      if (held.link?.isConnected) {
        // No listener holds clicks once the changes are saved: a plain click follows the link.
        if (clicks.current) clicks.current.follow(held);
        else held.link.click();
        return;
      }
      router.push(held.href);
    },
  };
}
