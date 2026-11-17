"use client";

import { Pause, Play } from "lucide-react";
import Image from "next/image";
import { useEffect, useRef, useState } from "react";
import { WindowFrame } from "./Screenshot";
import { FOCUS_RING } from "./site";
import { DEMO_CLIP } from "./shots";

// The muted demo loop. Nothing downloads until it is near the screen: the
// video has preload="none", and its still frame is a lazy, responsive image
// laid over it (a poster attribute would load one full-size file at once).
// The still shows until the clip first plays, and stays without script. The
// clip plays while at least 40% of it is visible, and never starts by itself
// for a visitor who asks for reduced motion. The button pauses and plays it,
// and a pause by the visitor holds until they play again.
export function DemoClip({ describedBy }: { describedBy: string }) {
  const video = useRef<HTMLVideoElement>(null);
  const pausedByVisitor = useRef(false);
  const [playing, setPlaying] = useState(false);
  const [started, setStarted] = useState(false);

  useEffect(() => {
    const v = video.current;
    if (!v) return;
    // React does not always write the muted attribute; browsers only autoplay muted video.
    v.muted = true;
    const reduce = window.matchMedia("(prefers-reduced-motion: reduce)");
    const onMotionChange = () => {
      if (reduce.matches) v.pause();
    };
    reduce.addEventListener("change", onMotionChange);
    if (typeof IntersectionObserver === "undefined") return () => reduce.removeEventListener("change", onMotionChange);
    const io = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting && !reduce.matches && !pausedByVisitor.current) v.play().catch(() => {});
        else if (!entry.isIntersecting) v.pause();
      },
      { threshold: 0.4 },
    );
    io.observe(v);
    return () => {
      io.disconnect();
      reduce.removeEventListener("change", onMotionChange);
    };
  }, []);

  const toggle = () => {
    const v = video.current;
    if (!v) return;
    if (v.paused) {
      pausedByVisitor.current = false;
      v.play().catch(() => {});
    } else {
      pausedByVisitor.current = true;
      v.pause();
    }
  };

  return (
    <div className="relative">
      <WindowFrame>
        <div className="grid">
          <video
            ref={video}
            muted
            loop
            playsInline
            preload="none"
            width={DEMO_CLIP.width}
            height={DEMO_CLIP.height}
            disablePictureInPicture
            aria-describedby={describedBy}
            onPlay={() => setPlaying(true)}
            onPlaying={() => setStarted(true)}
            onPause={() => setPlaying(false)}
            className="col-start-1 row-start-1 block aspect-video h-auto w-full bg-slate-950">
            <source src={DEMO_CLIP.webm} type="video/webm" />
            <source src={DEMO_CLIP.mp4} type="video/mp4" />
          </video>
          <Image
            src={DEMO_CLIP.poster}
            alt=""
            aria-hidden="true"
            width={DEMO_CLIP.width}
            height={DEMO_CLIP.height}
            sizes="(min-width: 1024px) 640px, calc(100vw - 32px)"
            className={`pointer-events-none col-start-1 row-start-1 block h-auto w-full transition-opacity duration-300 motion-reduce:transition-none ${started ? "opacity-0" : "opacity-100"}`}
          />
        </div>
      </WindowFrame>
      <button
        type="button"
        onClick={toggle}
        className={`absolute bottom-2.5 left-2.5 inline-flex min-h-8 items-center gap-1.5 rounded-full border border-white/15 bg-black/80 px-3 py-1 text-xs font-medium text-white transition-colors hover:bg-black motion-reduce:transition-none ${FOCUS_RING}`}>
        {playing ? <Pause className="h-3.5 w-3.5" aria-hidden="true" /> : <Play className="h-3.5 w-3.5" aria-hidden="true" />}
        {playing ? "Pause the demo" : "Play the demo"}
      </button>
    </div>
  );
}
