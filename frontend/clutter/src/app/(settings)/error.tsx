"use client";

import RouteError from "@/components/common/RouteError";

// An unexpected error on Connections or a settings page, shown in the app shell.
export default function SettingsError({ retry }: { error: unknown; retry: () => void }) {
  return <RouteError retry={retry} />;
}
