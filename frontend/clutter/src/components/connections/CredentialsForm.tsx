"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { RadioCard, RadioCardGroup, RadioCardTitle } from "@/components/ui/radio-card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Skeleton } from "@/components/ui/skeleton";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLook } from "@/lib/features/catalog/display";
import { useLinkedAccounts } from "@/lib/features/catalog/hooks";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { fitsCloudsYaml } from "@/lib/features/connections/cloudsYaml";
import { credentialHelp } from "@/lib/features/connections/display";
import { useConnections, useCreateConnection } from "@/lib/features/connections/hooks";
import { startRegion } from "@/lib/features/connections/linkedAccount";
import { offersLocalDevstack } from "@/lib/features/connections/localDevstack";
import { localConnectionName, NAME_MAX } from "@/lib/features/connections/localName";
import type { Connection } from "@/lib/features/connections/types";
import { isLocalDevApi } from "@/lib/features/organization/api";
import { cn } from "@/lib/utils";
import { ChevronDown, Loader2 } from "lucide-react";
import { useId, useState } from "react";
import BeforeYouStart from "./BeforeYouStart";
import CloudsYamlUpload from "./CloudsYamlUpload";
import { describedBy, Field, Notice } from "./Field";
import { UseLocalDevstack } from "./LocalConnect";

// What a cloud whose credentials are a clouds.yaml application credential
// (OpenStack) needs before it is connected. Add deployment pre-selects an
// image and server sizes from the cloud's own lists where DevStack's defaults
// are missing (connections/options.ts, applySuggestions).
const CLOUDS_YAML_BEFORE_YOU_START = [
  {
    lead: "You need:",
    text: "an application credential for the project Clutter should build in (in your cloud's dashboard: Identity, then Application Credentials). Give it an expiry date.",
  },
  { text: "Your project needs routers, floating IPs and volumes, and room for the servers you deploy." },
  { text: "When you deploy, Clutter picks an image and server sizes from your cloud's own lists. You can change them under Advanced." },
];

type Props = {
  token: string;
  organizationId: string;
  entry: CatalogEntry;
  onConnected: (connection: Connection) => void;
  // One column, for the side sheet.
  narrow?: boolean;
};

// CredentialsForm connects a cloud from pasted credentials, or from an
// account linked outside B: the form every connector gets from its credential
// spec. AWS, whose accounts are linked through A's CloudFormation flow, has
// AwsConnectForm instead.
export default function CredentialsForm({ token, organizationId, entry, onConnected, narrow = false }: Props) {
  const { manifest, credentials } = entry;
  const uid = useId();
  const regions = manifest.regions ?? [];
  const fields = credentials.fields ?? [];
  const linkedOnly = credentials.linked_only;
  const linkedQ = useLinkedAccounts(token, entry.has_linked_accounts ? organizationId : null, entry.has_linked_accounts ? manifest.connector : null);
  const connectionsQ = useConnections(token, organizationId);
  const createM = useCreateConnection(token);

  const [mode, setMode] = useState<"linked" | "credentials">(entry.has_linked_accounts || linkedOnly ? "linked" : "credentials");
  const [name, setName] = useState("");
  const [region, setRegion] = useState(() => startRegion(regions));
  const [values, setValues] = useState<Record<string, string>>({});
  const [linkedId, setLinkedId] = useState<string>("");
  const [errors, setErrors] = useState<Record<string, string>>({});

  // A clouds.yaml application credential (OpenStack): the form takes the file
  // and says what the cloud needs first.
  const cloudsYaml = fitsCloudsYaml(fields.map((f) => f.name));
  const entering = mode === "credentials" && !linkedOnly;
  const shortName = manifest.display?.short_name || manifest.title;
  const localDevstack = offersLocalDevstack(isLocalDevApi, fields.map((f) => f.name), mode, linkedOnly);
  // On a developer's machine the cloud running locally comes first, in one
  // click; the form for a cloud of one's own opens from a toggle.
  const [ownOpen, setOwnOpen] = useState(false);
  const showOwn = !localDevstack || ownOpen;
  const linkedAccounts = linkedQ.data ?? [];

  const clearError = (key: string) => setErrors((e) => (e[key] ? { ...e, [key]: "" } : e));

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    createM.reset();
    const found: Record<string, string> = {};
    const trimmed = name.trim();
    if (!trimmed) found.name = "Give the connection a name.";
    else if (trimmed.length > NAME_MAX) found.name = `Use at most ${NAME_MAX} characters.`;
    if (!region.trim()) found.region = "Choose a region.";
    if (mode === "linked" && !linkedId) found.linked = "Choose a linked account.";
    if (entering) {
      for (const f of fields) if (f.required && !(values[f.name] ?? "").trim()) found[`f:${f.name}`] = `Enter the ${f.label.toLowerCase()}.`;
    }
    setErrors(found);
    const first = Object.keys(found)[0];
    if (first) {
      document.getElementById(`${uid}-${first}`)?.focus();
      return;
    }
    const base = { connector: manifest.connector, name: trimmed, region: region.trim() };
    if (mode === "linked") {
      createM.mutate({ organizationId, data: { ...base, linked_id: linkedId } }, { onSuccess: onConnected });
      return;
    }
    const supplied: Record<string, string> = {};
    for (const f of fields) {
      const v = (values[f.name] ?? "").trim();
      if (v) supplied[f.name] = v;
    }
    createM.mutate(
      { organizationId, data: { ...base, credentials: supplied } },
      {
        onSuccess: (c) => {
          setValues({});
          onConnected(c);
        },
      },
    );
  };

  const id = (key: string) => `${uid}-${key}`;

  return (
    <section aria-label={`Connect ${manifest.title}`} className="space-y-5 animate-fade-in">
      <h3 className="text-foreground flex items-center gap-2.5 text-base font-semibold">
        <CloudMark cloud={cloudLook(manifest)} size={24} />
        Connect {manifest.title}
      </h3>

      {localDevstack && (
        <UseLocalDevstack
          token={token}
          organizationId={organizationId}
          connector={manifest.connector}
          cloudName={shortName}
          name={localConnectionName(shortName, (connectionsQ.data ?? []).map((c) => c.name))}
          onConnected={onConnected}
        />
      )}
      {localDevstack && (
        <button
          type="button"
          aria-expanded={ownOpen}
          onClick={() => setOwnOpen((o) => !o)}
          className="text-muted-foreground hover:text-foreground flex items-center gap-1 text-sm font-medium transition-colors motion-fast">
          <ChevronDown className={cn("size-4 transition-transform motion-fast", !ownOpen && "-rotate-90")} aria-hidden />
          I have my own {shortName}
        </button>
      )}

      {showOwn && (
        <form onSubmit={submit} noValidate className="space-y-5">
          {entry.has_linked_accounts && !linkedOnly && (
            <Tabs value={mode} onValueChange={(v) => setMode(v as "linked" | "credentials")}>
              <TabsList aria-label="How to connect">
                <TabsTrigger value="linked">Use a linked account</TabsTrigger>
                <TabsTrigger value="credentials">Enter the details</TabsTrigger>
              </TabsList>
            </Tabs>
          )}

          {entering && cloudsYaml && <BeforeYouStart items={CLOUDS_YAML_BEFORE_YOU_START} />}

          {mode === "linked" && (
            <div className="space-y-2">
              <p id={id("linked-label")} className="text-sm font-medium">
                Linked {manifest.title} accounts
              </p>
              {!entry.has_linked_accounts ? (
                <Notice tone="error">This Clutter server cannot read linked {manifest.title} accounts, so it cannot connect one.</Notice>
              ) : linkedQ.isLoading ? (
                <Skeleton className="h-14 rounded-xl" />
              ) : linkedQ.isError ? (
                <Notice tone="error">
                  Could not load linked accounts. {adapterErrorMessage(linkedQ.error)}
                  <Button type="button" size="xs" variant="outline" onClick={() => void linkedQ.refetch()}>
                    Retry
                  </Button>
                </Notice>
              ) : linkedAccounts.length === 0 ? (
                <Notice tone="info">No linked {manifest.title} accounts yet.</Notice>
              ) : (
                <RadioCardGroup
                  id={id("linked")}
                  aria-labelledby={id("linked-label")}
                  value={linkedId}
                  onValueChange={(v) => {
                    setLinkedId(v);
                    clearError("linked");
                    const a = linkedAccounts.find((x) => x.id === v);
                    if (a && !name.trim()) setName(a.name.slice(0, NAME_MAX));
                    if (a?.region) setRegion(a.region);
                  }}>
                  {linkedAccounts.map((a) => (
                    <RadioCard key={a.id} value={a.id} className="p-3">
                      <RadioCardTitle>{a.name}</RadioCardTitle>
                    </RadioCard>
                  ))}
                </RadioCardGroup>
              )}
              {errors.linked && <p className="text-danger text-xs">{errors.linked}</p>}
            </div>
          )}

          <div className={cn("grid gap-4", !narrow && "sm:grid-cols-2")}>
            <Field id={id("name")} label="Connection name" error={errors.name} hint="A name your team will recognise.">
              <Input
                id={id("name")}
                value={name}
                onChange={(e) => {
                  setName(e.target.value);
                  clearError("name");
                }}
                maxLength={NAME_MAX}
                placeholder="for example lab"
                autoComplete="off"
                aria-invalid={!!errors.name || undefined}
                aria-describedby={describedBy(id("name"), true, !!errors.name)}
              />
            </Field>
            <Field id={id("region")} label="Region" error={errors.region}>
              {regions.length > 0 ? (
                <Select
                  value={region}
                  onValueChange={(v) => {
                    setRegion(v);
                    clearError("region");
                  }}>
                  <SelectTrigger id={id("region")} className="w-full" aria-invalid={!!errors.region || undefined}>
                    <SelectValue placeholder="Choose a region" />
                  </SelectTrigger>
                  <SelectContent>
                    {regions.map((r) => (
                      <SelectItem key={r} value={r}>
                        {r}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              ) : (
                <Input
                  id={id("region")}
                  value={region}
                  onChange={(e) => {
                    setRegion(e.target.value);
                    clearError("region");
                  }}
                  maxLength={64}
                  placeholder="for example RegionOne"
                  autoComplete="off"
                  aria-invalid={!!errors.region || undefined}
                  aria-describedby={describedBy(id("region"), false, !!errors.region)}
                />
              )}
            </Field>
          </div>

          {entering && (
            <div className="space-y-4">
              {cloudsYaml && (
                <CloudsYamlUpload
                  onFill={(filled, cloudRegion, cloudName) => {
                    setErrors({});
                    setValues((v) => {
                      const next = { ...v };
                      for (const f of fields) if (filled[f.name]) next[f.name] = filled[f.name];
                      return next;
                    });
                    if (cloudRegion && (regions.length === 0 || regions.includes(cloudRegion))) setRegion(cloudRegion);
                    if (!name.trim()) setName(cloudName.slice(0, NAME_MAX));
                  }}
                />
              )}
              {credentials.kind === "stored" && (
                <p className="text-muted-foreground text-xs">Clutter stores these encrypted and never shows a secret again. It never asks for a cloud password.</p>
              )}
              {fields.map((f) => {
                const key = `f:${f.name}`;
                const help = credentialHelp(manifest.connector, f.name, f.help);
                return (
                  <Field key={f.name} id={id(key)} label={`${f.label}${f.required ? "" : " (optional)"}`} hint={help} error={errors[key]}>
                    <Input
                      id={id(key)}
                      type={f.secret ? "password" : "text"}
                      autoComplete={f.secret ? "new-password" : "off"}
                      spellCheck={false}
                      value={values[f.name] ?? ""}
                      onChange={(e) => {
                        setValues((v) => ({ ...v, [f.name]: e.target.value }));
                        clearError(key);
                      }}
                      className="font-mono"
                      aria-invalid={!!errors[key] || undefined}
                      aria-describedby={describedBy(id(key), !!help, !!errors[key])}
                    />
                  </Field>
                );
              })}
            </div>
          )}

          {createM.isError && <Notice tone="error">{adapterErrorMessage(createM.error)}</Notice>}

          <Button type="submit" disabled={createM.isPending} className={cn(narrow && "w-full")}>
            {createM.isPending ? (
              <>
                <Loader2 className="animate-spin" aria-hidden />
                Checking with {shortName}…
              </>
            ) : (
              "Check and connect"
            )}
          </Button>
        </form>
      )}
    </section>
  );
}
