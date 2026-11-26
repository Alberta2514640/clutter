"use client";

import { useRef } from "react";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { useMe } from "@/lib/features/user/hooks";
import { Notice } from "./Field";
import ConnectCloudFlow from "./ConnectCloudFlow";

type Props = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  // The cloud the sheet opens on, a connector id ("aws"). Without it, the
  // viewer's first cloud from onboarding, when this build offers it.
  preferredConnector?: string | null;
  // Called with the new connection's id once it is made and checked.
  onConnected?: (connectionId: string) => void;
};

// ConnectCloudSheet holds the one connection flow in a side sheet, so a cloud
// can be connected from anywhere (the Connections page, the editor's deploy
// panel, the first-deploy checklist) without leaving the page. The flow
// starts afresh each time the sheet opens.
export default function ConnectCloudSheet({ open, onOpenChange, organizationId, preferredConnector, onConnected }: Props) {
  const meQ = useMe();
  const token = meQ.data?.token ?? null;
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  // Every place opens the sheet from its own button, not a Radix trigger, so
  // Radix has nothing to give focus back to. The sheet keeps the element that
  // had focus as it opened and returns focus there when it closes, while that
  // element is still on the page.
  const opener = useRef<HTMLElement | null>(null);

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="gap-0 sm:max-w-xl"
        closeLabel="Close"
        onOpenAutoFocus={() => {
          opener.current = document.activeElement instanceof HTMLElement && document.activeElement !== document.body ? document.activeElement : null;
        }}
        onCloseAutoFocus={(e) => {
          const el = opener.current;
          opener.current = null;
          if (el?.isConnected) {
            e.preventDefault();
            el.focus();
          }
        }}>
        <SheetHeader className="border-b px-6 py-5">
          <SheetTitle className="text-lg tracking-tight">Connect a cloud</SheetTitle>
          <SheetDescription>A connection is a cloud account Clutter can build in. Every project in this team can deploy to it.</SheetDescription>
        </SheetHeader>
        <div className="px-6 py-6">
          {meQ.isLoading ? (
            <div role="status" aria-label="Loading" className="space-y-3">
              <Skeleton className="h-[74px] rounded-xl" />
              <Skeleton className="h-[74px] rounded-xl" />
            </div>
          ) : !token ? (
            <Notice tone="error">Sign in again to connect a cloud.</Notice>
          ) : (
            <ConnectCloudFlow
              token={token}
              organizationId={organizationId}
              initialConnector={preferredConnector ?? firstCloud}
              onConnected={(c) => onConnected?.(c.id)}
              onDone={() => onOpenChange(false)}
              narrow
            />
          )}
        </div>
      </SheetContent>
    </Sheet>
  );
}
