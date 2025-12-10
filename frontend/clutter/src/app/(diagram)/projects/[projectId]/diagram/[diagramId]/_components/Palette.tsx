"use client";

import React, { useMemo } from "react";
import type { PaletteItem } from "./types";

const DND_MIME = "application/x-palette-item";

export default function Palette() {
    const sections = useMemo(() => {
        const storage: PaletteItem[] = [
            { label: "DynamoDB", badge: "DB", category: "STORAGE" },
            { label: "S3", badge: "S3", category: "STORAGE" },
        ];

        const compute: PaletteItem[] = [
            { label: "Lambda", badge: "λ", category: "COMPUTE" },
            { label: "EC2 Container", badge: "EC2", category: "COMPUTE" }
        ];

        const network: PaletteItem[] = [{ label: "API Gateway", badge: "API", category: "NETWORK" }];

        return [
            { title: "COMPUTE", items: compute },
            { title: "STORAGE", items: storage },
            { title: "NETWORK", items: network },
        ] as const;
    }, []);

    return (
        <aside className="w-[270px] rounded-xl border border-white/10 bg-[rgba(20,25,35,0.6)] p-4 shadow-lg backdrop-blur-sm">
        <div className="mb-4 text-xs text-white/60">
            Drag components onto the canvas
            <br />
            Connect nodes by dragging from handles
        </div>

        <div className="space-y-4">
            {sections.map((sec) => (
            <div key={sec.title}>
                <div className="mb-2 text-[11px] font-semibold tracking-wider text-white/50">
                {sec.title}
                </div>

                <div className="space-y-2">
                {sec.items.map((item) => (
                    <div
                    key={`${sec.title}-${item.label}`}
                    draggable
                    onDragStart={(e) => {
                        e.dataTransfer.setData(DND_MIME, JSON.stringify(item));
                        e.dataTransfer.effectAllowed = "move";
                    }}
                    className="flex cursor-grab select-none items-center gap-3 rounded-lg border border-white/10 bg-white/5 px-3 py-2.5 transition-colors hover:border-white/20 hover:bg-white/10 active:cursor-grabbing"
                    >
                    <div className="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-white/15 bg-white/10 text-xs font-bold">
                        {item.badge}
                    </div>
                    <div className="text-sm font-medium text-white">{item.label}</div>
                    </div>
                ))}
                </div>
            </div>
            ))}
        </div>
        </aside>
    );
}