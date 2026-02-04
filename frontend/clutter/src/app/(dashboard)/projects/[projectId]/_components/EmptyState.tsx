"use client";

import { Button } from "@/components/ui/button";
import { ReactNode } from "react";

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
}

export function EmptyState({
  icon,
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="w-full flex justify-center">
      <div
        className="
          w-full max-w-4xl 
          text-center 
          py-24 
          mt-12
          rounded-2xl
          border border-dashed border-slate-600/50
          bg-slate-900/20
        "
      >
        <div className="text-5xl mb-6">{icon}</div>
        <h2 className="text-2xl font-semibold text-white mb-3">{title}</h2>
        <p className="text-gray-400 mb-8">{description}</p>
        
        {actionLabel && onAction && (
          <Button
            onClick={onAction}
            className="bg-slate-800 border border-slate-700 hover:border-teal-500/50 text-gray-300 px-5 py-4 rounded-lg"
          >
            {actionLabel}
          </Button>
        )}
      </div>
    </div>
  );
}