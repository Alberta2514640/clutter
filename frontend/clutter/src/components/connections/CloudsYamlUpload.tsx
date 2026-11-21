"use client";

import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { APP_CRED_HELP, MAX_CLOUDS_FILE, readCloudsFiles, splitCloudsFiles, type CloudsEntry } from "@/lib/features/connections/cloudsYaml";
import { cn } from "@/lib/utils";
import { FileUp, Loader2 } from "lucide-react";
import { useId, useState } from "react";
import { Notice } from "./Field";

// CloudsYamlUpload fills the connection form from the clouds.yaml (and
// secure.yaml) an OpenStack dashboard downloads. The files are read in the
// browser only; onFill gets the form's own fields, and the page sends nothing
// else. Nothing is kept once the form is filled.
export default function CloudsYamlUpload({
  onFill,
}: {
  onFill: (values: Record<string, string>, region: string, cloudName: string) => void;
}) {
  const inputId = useId();
  const pickId = useId();
  const [reading, setReading] = useState(false);
  const [dragging, setDragging] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [entries, setEntries] = useState<CloudsEntry[] | null>(null);
  const [pick, setPick] = useState("");
  const [filled, setFilled] = useState<{ name: string; missing: string[] } | null>(null);

  const fillFrom = (e: CloudsEntry) => {
    if (!e.ok) {
      setError(e.message);
      return;
    }
    onFill(e.values, e.region, e.name);
    setFilled({ name: e.name, missing: e.missing });
    setEntries(null);
    setError(null);
  };

  const readFiles = async (list: FileList | null) => {
    setError(null);
    setFilled(null);
    setEntries(null);
    const files = Array.from(list ?? []);
    if (files.length === 0) return;
    const split = splitCloudsFiles(files);
    if (!split.ok) return setError(split.message);
    if (files.some((f) => f.size > MAX_CLOUDS_FILE)) return setError("This file is too large to be a clouds.yaml.");
    setReading(true);
    let cloudsText: string;
    let secureText: string | null = null;
    try {
      cloudsText = await split.clouds.text();
      if (split.secure) secureText = await split.secure.text();
    } catch {
      setReading(false);
      return setError("Could not read the file. Try adding it again.");
    }
    setReading(false);
    const r = readCloudsFiles(cloudsText, secureText);
    if (!r.ok) return setError(r.message);
    if (r.entries.length === 1) return fillFrom(r.entries[0]);
    setEntries(r.entries);
    setPick((r.entries.find((e) => e.ok) ?? r.entries[0]).name);
  };

  return (
    <div className="space-y-2">
      <div
        onDragOver={(e) => {
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setDragging(false);
          void readFiles(e.dataTransfer.files);
        }}
        className={cn(
          "flex flex-col gap-3 rounded-lg border border-dashed px-4 py-3.5 text-sm transition-colors motion-fast sm:flex-row sm:items-center sm:justify-between",
          dragging ? "border-primary bg-primary/5" : "bg-muted/30",
        )}>
        <span className="text-muted-foreground">
          Have a <span className="text-foreground font-mono">clouds.yaml</span>? Drop it here (with its{" "}
          <span className="text-foreground font-mono">secure.yaml</span>, if you have one) to fill the fields below. It is read in your browser and never
          uploaded.
        </span>
        <label
          htmlFor={inputId}
          className="border-input bg-background hover:bg-accent focus-within:ring-ring/50 inline-flex h-8 shrink-0 cursor-pointer items-center gap-1.5 self-start rounded-md border px-3 text-sm font-medium shadow-xs transition-colors focus-within:ring-[3px] sm:self-auto">
          {reading ? <Loader2 className="size-4 animate-spin" aria-hidden /> : <FileUp className="size-4" aria-hidden />}
          {reading ? "Reading…" : "Choose files"}
          <input
            id={inputId}
            type="file"
            multiple
            accept=".yaml,.yml,application/yaml,text/yaml"
            className="sr-only"
            onChange={(e) => {
              void readFiles(e.target.files);
              e.target.value = "";
            }}
          />
        </label>
      </div>

      {entries && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
          <div className="flex-1 space-y-1.5">
            <label htmlFor={pickId} className="text-sm font-medium">
              This file lists {entries.length} clouds. Which one?
            </label>
            <Select value={pick} onValueChange={setPick}>
              <SelectTrigger id={pickId} className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {entries.map((e) => (
                  <SelectItem key={e.name} value={e.name}>
                    {e.name}
                    {e.ok ? "" : " (cannot be used)"}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <Button
            type="button"
            variant="secondary"
            onClick={() => {
              const e = entries.find((x) => x.name === pick);
              if (e) fillFrom(e);
            }}>
            Use this cloud
          </Button>
        </div>
      )}

      {error && <Notice tone="error">{error}</Notice>}
      {filled &&
        (filled.missing.length === 0 ? (
          <Notice tone="success">Filled from {filled.name}. Check the name and region, then choose Check and connect.</Notice>
        ) : (
          <Notice tone="info">
            Filled from {filled.name}. It does not hold the {filled.missing.join(", ")}; enter {filled.missing.length === 1 ? "it" : "them"} below.
            {filled.missing.includes("application credential secret") && " The secret is often in secure.yaml; add both files together."}
          </Notice>
        ))}
      {!error && !filled && !entries && <p className="text-muted-foreground text-xs">No application credential yet? {APP_CRED_HELP}</p>}
    </div>
  );
}
