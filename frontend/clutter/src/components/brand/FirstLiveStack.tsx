"use client";

import { usePreferencesStore } from "@/lib/features/preferences/uiStore";
import { cn } from "@/lib/utils";
import { useEffect, useState } from "react";
import styles from "./brand.module.css";
import IsoLoader from "./IsoLoader";

const reducedMotion = () => typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

// FirstLiveStack marks the first time, in this browser, that a deployment the
// viewer is watching turns Live: the logo's blocks drop in beside the state,
// hold, and fade, in about a second. A deployment already Live when the page
// opens does not play it. Nothing shows under reduced motion. Decorative.
export default function FirstLiveStack({ live, className }: { live: boolean; className?: string }) {
  const shown = usePreferencesStore((s) => s.firstLiveShown);
  const markShown = usePreferencesStore((s) => s.markFirstLiveShown);
  const [wasLive, setWasLive] = useState(live);
  const [playing, setPlaying] = useState(false);

  // A change of `live` since the last render (React's "adjust state when a
  // prop changes" pattern): only a turn to Live plays.
  if (live !== wasLive) {
    setWasLive(live);
    if (live && !shown && !reducedMotion()) setPlaying(true);
  }

  useEffect(() => {
    if (playing) markShown();
  }, [playing, markShown]);

  if (!playing) return null;
  return (
    <span
      aria-hidden="true"
      data-testid="first-live-stack"
      className={cn(styles.flourish, "pointer-events-none inline-flex", className)}
      onAnimationEnd={(event) => {
        if (event.target === event.currentTarget) setPlaying(false);
      }}>
      <IsoLoader mode="once" size={22} />
    </span>
  );
}
