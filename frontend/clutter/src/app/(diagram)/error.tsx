"use client";

import RouteError from "@/components/common/RouteError";

// An unexpected error in the editor. The layout around it is full screen, so
// the message is centred.
export default function DiagramError({ retry }: { error: unknown; retry: () => void }) {
  return (
    <div className="grid h-full place-items-center px-4">
      <div className="w-full max-w-md">
        <RouteError retry={retry} />
      </div>
    </div>
  );
}
