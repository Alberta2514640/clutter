"use client";
import React, { createContext, useMemo, useEffect } from "react";
import * as Y from "yjs";
import { WebsocketProvider } from "y-websocket";
export const YCtx = createContext<{ doc: Y.Doc; provider: WebsocketProvider } | null>(null);
export function YProvider({ room, children }: { room: string; children: React.ReactNode }) {
  const doc = useMemo(() => new Y.Doc(), []);
  const provider = useMemo(() => new WebsocketProvider(process.env.NEXT_PUBLIC_YWS || "ws://localhost:1234", room, doc), [room, doc]);
  useEffect(() => () => provider.destroy(), [provider]);
  return <YCtx.Provider value={{ doc, provider }}>{children}</YCtx.Provider>;
}
