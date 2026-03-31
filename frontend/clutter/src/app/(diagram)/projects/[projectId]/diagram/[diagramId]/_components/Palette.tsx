"use client";

import { ChevronLeft, ChevronRight, Layers } from "lucide-react";
import Image from "next/image";
import { useMemo, useState } from "react";

const DND_MIME = "application/x-palette-item";

export type PaletteItem = {
  type: "lambda" | "ec2" | "dynamodb" | "s3" | "apigateway";
  label: string;
  img: string;
};

type Section = { title: string; items: PaletteItem[] };

type PaletteProps = {
  isReadOnly?: boolean;
};

export default function Palette({ isReadOnly = false }: PaletteProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  const sections = useMemo<Section[]>(() => {
    const storage: PaletteItem[] = [
      { type: "dynamodb", label: "DynamoDB", img: "/aws/dynamodb.png" },
      { type: "s3", label: "S3", img: "/aws/s3.png" },
    ];

    const compute: PaletteItem[] = [
      { type: "lambda", label: "Lambda", img: "/aws/lambda.png" },
      { type: "ec2", label: "EC2 Container", img: "/aws/ec2.png" },
    ];

    const network: PaletteItem[] = [
      { type: "apigateway", label: "API Gateway", img: "/aws/apigateway.png" },
    ];

    return [
      { title: "COMPUTE", items: compute },
      { title: "STORAGE", items: storage },
      { title: "NETWORK", items: network },
    ];
  }, []);

  return (
    <aside
      className={[
        "relative z-50 overflow-visible pointer-events-auto h-full shrink-0 border-r border-slate-800 bg-slate-950/70 backdrop-blur",
        "transition-[width] duration-200",
        isCollapsed ? "w-14" : "w-[200px]",
      ].join(" ")}
    >
      <button
        type="button"
        onClick={(e) => {
          e.stopPropagation();
          setIsCollapsed((v) => !v);
        }}
        className={[
          "absolute -right-3 top-1/2 -translate-y-1/2 z-[60] pointer-events-auto",
          "grid h-7 w-7 place-items-center rounded-full",
          "border border-slate-700 bg-slate-900 shadow-lg transition hover:bg-slate-800",
        ].join(" ")}
        aria-label={isCollapsed ? "Expand palette" : "Collapse palette"}
        title={isCollapsed ? "Expand" : "Collapse"}
      >
        {isCollapsed ? (
          <ChevronRight className="h-4 w-4 text-slate-300" />
        ) : (
          <ChevronLeft className="h-4 w-4 text-slate-300" />
        )}
      </button>

      <div className="flex h-14 items-center gap-2 border-b border-slate-800 px-3">
        <div className="flex min-w-0 items-center gap-2">
          <div className="grid h-8 w-8 place-items-center rounded-lg border border-slate-800 bg-slate-900">
            <Layers className="h-4 w-4 text-slate-300" />
          </div>

          {!isCollapsed && (
            <div className="min-w-0">
              <div className="truncate text-sm font-semibold text-slate-100">
                Resources
              </div>
            </div>
          )}
        </div>
      </div>

      {isCollapsed ? (
        <div className="flex h-[calc(100%-3.5rem)] items-center justify-center">
          <span className="select-none text-[11px] font-semibold tracking-widest text-slate-300 [writing-mode:vertical-rl] rotate-180">
            PALETTE
          </span>
        </div>
      ) : (
        <div className="h-[calc(100%-3.5rem)] overflow-y-auto px-3 py-3">
          <div className="space-y-5">
            {sections.map((sec) => (
              <section key={sec.title}>
                <div className="mb-2 flex items-center gap-2">
                  <div className="text-[11px] font-semibold tracking-wider text-slate-400">
                    {sec.title}
                  </div>
                  <div className="h-px flex-1 bg-slate-800" />
                </div>

                <div className="space-y-2">
                  {sec.items.map((item) => (
                    <div
                      key={`${sec.title}-${item.label}`}
                      draggable={!isReadOnly}
                      onDragStart={(e) => {
                        if (isReadOnly) {
                          e.preventDefault();
                          return;
                        }
                        e.dataTransfer.setData(DND_MIME, JSON.stringify(item));
                        e.dataTransfer.effectAllowed = "move";
                      }}
                      className={[
                        "flex select-none items-center gap-3 rounded-lg",
                        "border border-slate-800 bg-slate-900/40 px-3 py-2.5 transition",
                        isReadOnly
                          ? "cursor-not-allowed opacity-50"
                          : "cursor-grab hover:bg-slate-900/70 hover:border-slate-700 active:cursor-grabbing",
                      ].join(" ")}
                      title={
                        isReadOnly
                          ? "Palette disabled during deployment"
                          : "Drag to add"
                      }
                    >
                      <div className="grid h-8 w-8 shrink-0 place-items-center rounded-md border border-slate-800 bg-slate-950">
                        <Image
                          src={item.img}
                          alt={item.label}
                          width={22}
                          height={22}
                          className="object-contain"
                          unoptimized
                        />
                      </div>
                      <div className="min-w-0">
                        <div className="truncate text-sm font-medium text-slate-100">
                          {item.label}
                        </div>
                        <div className="truncate text-[11px] text-slate-400">
                          {isReadOnly ? "Temporarily disabled" : "Drag to add"}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              </section>
            ))}
          </div>
        </div>
      )}
    </aside>
  );
}