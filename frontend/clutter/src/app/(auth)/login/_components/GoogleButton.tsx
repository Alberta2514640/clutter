"use client";

import { Skeleton } from "@/components/ui/skeleton";
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google";
import { useEffect, useRef, useState } from "react";

// Google draws its own button in an iframe at a pixel width (at most 400) and
// its own height (40px for "large"). This measures the form column so the
// button fills it, inside a 44px slot so the layout does not move while
// Google's script loads.
const GOOGLE_MAX_WIDTH = 400;

type Props = {
  onCredential: (response: CredentialResponse) => void;
  onError: () => void;
};

export default function GoogleButton({ onCredential, onError }: Props) {
  const slotRef = useRef<HTMLDivElement>(null);
  const [width, setWidth] = useState<number | null>(null);

  useEffect(() => {
    const el = slotRef.current;
    if (!el) return;
    const measure = () => setWidth(Math.min(GOOGLE_MAX_WIDTH, Math.floor(el.getBoundingClientRect().width)));
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return (
    <div ref={slotRef} className="relative flex h-11 w-full items-center justify-center">
      {width === null ? (
        <Skeleton className="h-10 w-full" />
      ) : (
        // A new width draws a new button; Google does not resize one.
        <GoogleLogin
          key={width}
          onSuccess={onCredential}
          onError={onError}
          theme="outline"
          size="large"
          text="continue_with"
          shape="rectangular"
          logo_alignment="center"
          width={width}
        />
      )}
    </div>
  );
}
