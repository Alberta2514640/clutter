"use client";

import CloudMark from "@/components/clouds/CloudMark";
import { Button } from "@/components/ui/button";
import { adapterErrorMessage } from "@/lib/adapterApi";
import { cloudLook, cloudTitleWithNote } from "@/lib/features/catalog/display";
import { useLinkedAccounts } from "@/lib/features/catalog/hooks";
import type { CatalogEntry, LinkedAccount } from "@/lib/features/catalog/types";
import { fitsCloudsYaml } from "@/lib/features/connections/cloudsYaml";
import { useConnections, useCreateConnection } from "@/lib/features/connections/hooks";
import { chosenLinkedAccount, startRegion } from "@/lib/features/connections/linkedAccount";
import { offersLocalDevstack } from "@/lib/features/connections/localDevstack";
import { localConnectionName } from "@/lib/features/connections/localName";
import { DEFAULT_AWS_REGION, isLocalDevApi } from "@/lib/features/organization/api";
import { ChevronDown, ChevronRight, Link2, Loader2, Plus } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import AwsLinkFlow from "../../aws-account/_components/AwsLinkFlow";
import CloudsYamlUpload from "./CloudsYamlUpload";
import ConnectLocalAccount from "./ConnectLocalAccount";
import { Loading, Notice } from "./ConnectionsPage";
import UseLocalDevstack from "./UseLocalDevstack";

// The account-link flow of a cloud linked outside B, shown inline here, the
// page that lists and removes its links, and the region the flow opens the
// cloud in, which a new connection starts on. AWS links through A's
// CloudFormation flow (AwsLinkFlow).
// local: on a developer's machine, A2 links a fake account of this cloud
// (ADR-011), so Connect local can link and connect it in one step.
const LINK_FLOWS: Record<string, { page: string; pageLabel: string; region: string; local: boolean }> = {
  aws: { page: "/settings/organization/aws-account", pageLabel: "Manage AWS account links", region: DEFAULT_AWS_REGION, local: true },
};

const inputClass =
  "w-full rounded-lg border border-slate-700 bg-slate-900/60 px-3 py-2 text-sm text-white outline-none focus:border-teal-500/60";

type Props = { token: string; organizationId: string; catalog: CatalogEntry[] };

export default function AddConnection({ token, organizationId, catalog }: Props) {
  const [connector, setConnector] = useState<string | null>(null);
  // The name of the connection just added.
  const [added, setAdded] = useState<string | null>(null);
  const entry = catalog.find((c) => c.manifest.connector === connector) ?? null;

  return (
    <div className="space-y-4">
      <div role="radiogroup" aria-label="Cloud" className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {catalog.map((c) => {
          const selected = c.manifest.connector === connector;
          return (
            <button
              key={c.manifest.connector}
              type="button"
              role="radio"
              aria-checked={selected}
              onClick={() => {
                setConnector(c.manifest.connector);
                setAdded(null);
              }}
              className={`flex items-center gap-3 rounded-xl border px-4 py-3 text-left transition ${
                selected ? "border-teal-500 bg-teal-500/10" : "border-slate-800 bg-slate-900/50 hover:border-slate-600"
              }`}>
              <CloudMark cloud={cloudLook(c.manifest)} size={28} />
              <span>
                <span className="block font-medium text-white">{cloudTitleWithNote(cloudLook(c.manifest))}</span>
                <span className="block text-xs text-slate-400">
                  {c.credentials.linked_only
                    ? "Link your account, or use one you linked"
                    : c.has_linked_accounts
                      ? "Use an account you linked, or enter its details"
                      : c.credentials.kind === "borrowed"
                        ? "Borrowed access"
                        : "Credentials stored encrypted"}
                </span>
              </span>
            </button>
          );
        })}
      </div>

      {added && (
        <Notice tone="success">
          Connected {added}. Clutter checked it and it works.
        </Notice>
      )}

      {entry && (
        <ConnectionForm
          key={entry.manifest.connector}
          token={token}
          organizationId={organizationId}
          entry={entry}
          onAdded={(connectionName) => {
            setAdded(connectionName);
            setConnector(null);
          }}
        />
      )}
    </div>
  );
}

function ConnectionForm({
  token,
  organizationId,
  entry,
  onAdded,
}: {
  token: string;
  organizationId: string;
  entry: CatalogEntry;
  onAdded: (connectionName: string) => void;
}) {
  const { manifest, credentials } = entry;
  const regions = manifest.regions ?? [];
  const linkedQ = useLinkedAccounts(token, entry.has_linked_accounts ? organizationId : null, entry.has_linked_accounts ? manifest.connector : null);
  const createM = useCreateConnection(token);
  const linkFlow = LINK_FLOWS[manifest.connector];

  // A linked-only cloud (AWS) is connected only from an account this
  // organization linked; B refuses pasted credentials for it.
  const linkedOnly = credentials.linked_only;
  const [mode, setMode] = useState<"linked" | "credentials">(entry.has_linked_accounts || linkedOnly ? "linked" : "credentials");
  const [name, setName] = useState("");
  const [region, setRegion] = useState(() => startRegion(regions, linkFlow?.region));
  const [values, setValues] = useState<Record<string, string>>({});
  const [picked, setPicked] = useState<LinkedAccount | null>(null);
  // The name of an account just linked inline; it is chosen once B lists it.
  const [justLinked, setJustLinked] = useState<string | null>(null);
  const [linkOpen, setLinkOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const linkedAccounts = linkedQ.data ?? [];
  const linked = chosenLinkedAccount(linkedAccounts, picked, justLinked);
  const fields = credentials.fields ?? [];

  // On a developer's machine the cloud running locally comes first, in one
  // click; the form for a cloud of one's own sits behind a toggle.
  const connectionsQ = useConnections(token, organizationId);
  const shortName = manifest.display?.short_name || manifest.title;
  const localName = localConnectionName(shortName, (connectionsQ.data ?? []).map((c) => c.name));
  const localDevstack = offersLocalDevstack(isLocalDevApi, fields.map((f) => f.name), mode, linkedOnly);
  const localAccount = isLocalDevApi && linkedOnly && !!linkFlow?.local;
  const localFirst = localDevstack || localAccount;
  const [ownOpen, setOwnOpen] = useState(false);
  const showOwn = !localFirst || ownOpen;
  // With no linked account yet, linking one is the only way forward, so the flow is open.
  const showLinkFlow = !!linkFlow && linkedQ.isSuccess && (linkOpen || linkedAccounts.length === 0);
  // The flow closes once the new account is listed, taking its own notice
  // with it, so the form says what happened.
  const justLinkedAccount = !picked && !showLinkFlow && justLinked ? linked : null;
  const choose = (a: LinkedAccount) => {
    setPicked(a);
    if (!name.trim()) setName(a.name.slice(0, 64));
    if (a.region) setRegion(a.region);
  };

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    const trimmed = name.trim();
    if (!trimmed) return setFormError("Give the connection a name.");
    if (trimmed.length > 64) return setFormError("The name must be at most 64 characters.");
    if (!region.trim()) return setFormError("Choose a region.");
    if (mode === "linked") {
      if (!linked) return setFormError("Choose a linked account.");
      createM.mutate(
        { organizationId, data: { connector: manifest.connector, name: trimmed, region: region.trim(), linked_id: linked.id } },
        { onSuccess: (c) => onAdded(c.name) },
      );
      return;
    }
    const missing = fields.find((f) => f.required && !(values[f.name] ?? "").trim());
    if (missing) return setFormError(`${missing.label} is required.`);
    const supplied: Record<string, string> = {};
    for (const f of fields) {
      const v = (values[f.name] ?? "").trim();
      if (v) supplied[f.name] = v;
    }
    createM.mutate(
      { organizationId, data: { connector: manifest.connector, name: trimmed, region: region.trim(), credentials: supplied } },
      {
        onSuccess: (c) => {
          setValues({});
          onAdded(c.name);
        },
      },
    );
  };

  return (
    // The account-link flow holds forms of its own, so only the connection's
    // own fields sit in this form, below it.
    <section className="space-y-5 rounded-xl border border-slate-800 bg-slate-900/40 p-5" aria-label={`Connect ${manifest.title}`}>
      <h4 className="flex items-center gap-2 font-semibold text-white">
        <CloudMark cloud={cloudLook(manifest)} size={24} />
        Connect {manifest.title}
      </h4>

      {localDevstack && (
        <UseLocalDevstack token={token} organizationId={organizationId} connector={manifest.connector} cloudName={shortName} name={localName} onLinked={onAdded} />
      )}
      {localAccount && linkFlow && (
        <ConnectLocalAccount
          token={token}
          organizationId={organizationId}
          connector={manifest.connector}
          cloudName={shortName}
          name={localName}
          region={linkFlow.region}
          onAdded={onAdded}
        />
      )}
      {localFirst && (
        <button
          type="button"
          aria-expanded={ownOpen}
          onClick={() => setOwnOpen((o) => !o)}
          className="flex items-center gap-1 text-sm text-slate-300 hover:text-white">
          {ownOpen ? <ChevronDown className="h-4 w-4" aria-hidden /> : <ChevronRight className="h-4 w-4" aria-hidden />}
          {linkedOnly ? `I have my own ${shortName} account` : `I have my own ${shortName}`}
        </button>
      )}

      {showOwn && entry.has_linked_accounts && !linkedOnly && (
        <div className="flex gap-2" role="tablist" aria-label="How to connect">
          {(["linked", "credentials"] as const).map((m) => (
            <button
              key={m}
              type="button"
              role="tab"
              aria-selected={mode === m}
              onClick={() => setMode(m)}
              className={`rounded-lg px-3 py-1.5 text-sm ${mode === m ? "bg-teal-600 text-white" : "bg-slate-800 text-slate-300 hover:bg-slate-700"}`}>
              {m === "linked" ? "Use a linked account" : "Enter the details"}
            </button>
          ))}
        </div>
      )}

      {showOwn && mode === "linked" && (
        <div className="space-y-2">
          {linkedOnly && <p className="text-sm text-slate-300">Use a linked {manifest.title} account.</p>}
          {linkedOnly && !entry.has_linked_accounts ? (
            <Notice tone="error">This Clutter server cannot read linked {manifest.title} accounts, so it cannot connect one.</Notice>
          ) : linkedQ.isLoading ? (
            <Loading text="Loading linked accounts…" />
          ) : linkedQ.isError ? (
            <Notice tone="error">Could not load linked accounts: {adapterErrorMessage(linkedQ.error)}</Notice>
          ) : linkedAccounts.length === 0 ? (
            <p className="text-sm text-slate-300">
              {linkFlow ? `No ${manifest.title} account is linked yet. Link one here first, then connect it.` : `No linked ${manifest.title} accounts yet.`}
            </p>
          ) : (
            <ul className="space-y-2">
              {linkedAccounts.map((a) => (
                <li key={a.id}>
                  <button
                    type="button"
                    onClick={() => choose(a)}
                    aria-pressed={linked?.id === a.id}
                    className={`flex w-full items-center gap-2 rounded-lg border px-3 py-2 text-left text-sm ${
                      linked?.id === a.id ? "border-teal-500 bg-teal-500/10 text-white" : "border-slate-700 bg-slate-900/60 text-slate-200 hover:border-slate-500"
                    }`}>
                    <Link2 className="h-4 w-4 text-teal-300" aria-hidden />
                    Use {a.name}
                  </button>
                </li>
              ))}
            </ul>
          )}
          {justLinkedAccount && (
            <Notice tone="success">
              {justLinkedAccount.name} is linked and chosen above. Check the name of the connection, then choose Check and connect.
            </Notice>
          )}
          {showLinkFlow && (
            <div className="space-y-3 rounded-xl border border-slate-800 bg-slate-950/40 p-4">
              <h5 className="text-sm font-semibold text-white">Link a new {manifest.title} account</h5>
              <AwsLinkFlow
                token={token}
                organizationId={organizationId}
                wide="xl"
                onLinked={(accountName) => {
                  setJustLinked(accountName);
                  setPicked(null);
                  setLinkOpen(false);
                  if (!name.trim()) setName(accountName.slice(0, 64));
                }}
                afterLink={<p>It is chosen above. Check the name of the connection, then choose Check and connect.</p>}
              />
            </div>
          )}
          {linkFlow && (
            <div className="flex flex-wrap items-center gap-x-5 gap-y-2">
              {linkedQ.isSuccess && linkedAccounts.length > 0 && !linkOpen && (
                <button type="button" onClick={() => setLinkOpen(true)} className="inline-flex items-center gap-1 text-sm text-teal-300 hover:underline">
                  <Plus className="h-3.5 w-3.5" aria-hidden />
                  Link a new {manifest.title} account
                </button>
              )}
              <Link href={linkFlow.page} className="inline-block text-xs text-slate-400 underline underline-offset-2 hover:text-white">
                {linkFlow.pageLabel}
              </Link>
            </div>
          )}
        </div>
      )}

      {showOwn && (
      <form onSubmit={submit} className="space-y-5">
      <div className="grid gap-4 sm:grid-cols-2">
        <label className="space-y-1 text-sm text-slate-300">
          <span>Connection name</span>
          <input value={name} onChange={(e) => setName(e.target.value)} maxLength={64} placeholder="for example lab" className={inputClass} />
        </label>
        <label className="space-y-1 text-sm text-slate-300">
          <span>Region</span>
          {regions.length > 0 ? (
            <select value={region} onChange={(e) => setRegion(e.target.value)} className={inputClass}>
              {regions.map((r) => (
                <option key={r} value={r}>
                  {r}
                </option>
              ))}
            </select>
          ) : (
            <input value={region} onChange={(e) => setRegion(e.target.value)} maxLength={64} placeholder="for example RegionOne" className={inputClass} />
          )}
        </label>
      </div>

      {mode === "credentials" && !linkedOnly && (
        <div className="space-y-4">
          {fitsCloudsYaml(fields.map((f) => f.name)) && (
            <CloudsYamlUpload
              onFill={(filled, cloudRegion, cloudName) => {
                setFormError(null);
                setValues((v) => {
                  const next = { ...v };
                  for (const f of fields) if (filled[f.name]) next[f.name] = filled[f.name];
                  return next;
                });
                if (cloudRegion && (regions.length === 0 || regions.includes(cloudRegion))) setRegion(cloudRegion);
                if (!name.trim()) setName(cloudName.slice(0, 64));
              }}
            />
          )}
          {credentials.kind === "stored" && (
            <p className="text-xs text-slate-400">Clutter stores these encrypted and never shows a secret again. It never asks for a cloud password.</p>
          )}
          {fields.map((f) => (
            <label key={f.name} className="block space-y-1 text-sm text-slate-300">
              <span>
                {f.label}
                {f.required ? "" : " (optional)"}
              </span>
              <input
                type={f.secret ? "password" : "text"}
                autoComplete={f.secret ? "new-password" : "off"}
                spellCheck={false}
                value={values[f.name] ?? ""}
                onChange={(e) => setValues((v) => ({ ...v, [f.name]: e.target.value }))}
                className={`${inputClass} font-mono`}
              />
              {f.help && <span className="block text-xs text-slate-500">{f.help}</span>}
            </label>
          ))}
        </div>
      )}

      {(formError || createM.isError) && <Notice tone="error">{formError ?? adapterErrorMessage(createM.error)}</Notice>}

      <Button type="submit" disabled={createM.isPending} className="bg-teal-600 text-white hover:bg-teal-500">
        {createM.isPending ? (
          <>
            <Loader2 className="animate-spin" />
            Checking with {manifest.title}…
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
