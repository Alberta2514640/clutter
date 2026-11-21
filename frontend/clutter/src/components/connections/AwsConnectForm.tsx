"use client";

import CloudMark from "@/components/clouds/CloudMark";
import ConfirmDialog from "@/components/common/ConfirmDialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { toast } from "@/components/ui/sonner";
import { cloudLook } from "@/lib/features/catalog/display";
import type { CatalogEntry } from "@/lib/features/catalog/types";
import { accountsInOrder, connectFailure, isFakeLocalRole, resumeTarget, roleArnProblem } from "@/lib/features/connections/awsLink";
import { useConnections, useCreateConnection } from "@/lib/features/connections/hooks";
import { startRegion } from "@/lib/features/connections/linkedAccount";
import { localConnectionName, NAME_MAX } from "@/lib/features/connections/localName";
import type { Connection } from "@/lib/features/connections/types";
import { DEFAULT_AWS_REGION } from "@/lib/features/organization/api";
import {
  useCreateCloudFormationStackUrl,
  useDeleteOrganizationAccount,
  useOrganizationAccounts,
  useUpdateOrganizationAccount,
} from "@/lib/features/organization/hooks";
import { AWS_ACCOUNT_NAME_MAX, awsAccountNameHint, awsAccountNameProblem } from "@/lib/features/organization/rules";
import type { CloudFormationStackUrlResponse, OrganizationAwsAccount } from "@/lib/features/organization/types";
import { ApiError, apiErrorMessage } from "@/lib/features/user/session";
import { cn } from "@/lib/utils";
import { Check, ExternalLink, Loader2, Trash2 } from "lucide-react";
import { useId, useState, type ReactNode } from "react";
import BeforeYouStart from "./BeforeYouStart";
import { describedBy, Field, Notice } from "./Field";
import { ConnectLocalAccount } from "./LocalConnect";

const AWS_SETUP_GUIDE_URL = "/guide/AWS_Account_Link_Steps.pdf";

// What the CloudFormation stack (client_side_terraform_deployer_role.yaml)
// needs and creates, and what its role may do (ADR-027). Change this with the
// stack.
const AWS_BEFORE_YOU_START = [
  { lead: "You need:", text: "to be signed in to AWS as someone who can create a CloudFormation stack that makes IAM roles (an administrator)." },
  { lead: "What it creates:", text: "one role Clutter uses, the policies that limit it, and limits on anything Clutter creates. It is free." },
  {
    lead: "What Clutter can do:",
    text: "it cannot change servers, functions, tables or APIs it did not create. The exceptions: it can delete an empty bucket, change a bucket's tags, versioning and public access settings, and detach an Elastic IP address. It never asks for access keys, and its keys last at most one hour.",
  },
  { lead: "To remove its access:", text: "destroy your deployments, then delete the stack." },
];

type Props = {
  token: string;
  organizationId: string;
  entry: CatalogEntry;
  onConnected: (connection: Connection) => void;
  // One column, for the side sheet.
  narrow?: boolean;
};

// The link Clutter opened in AWS during this visit, with the name it was given.
type Started = CloudFormationStackUrlResponse & { name: string };

// AwsConnectForm is the one way to connect AWS: name the account, open AWS to
// create Clutter's role, paste the role ARN, and Check and connect links the
// account and connects it in one step. A half-finished link is resumed or
// removed here, and an account linked earlier is connected from its row.
export default function AwsConnectForm({ token, organizationId, entry, onConnected, narrow = false }: Props) {
  const { manifest } = entry;
  const uid = useId();
  const id = (key: string) => `${uid}-${key}`;
  const regions = manifest.regions ?? [];
  const shortName = manifest.display?.short_name || manifest.title;

  const accountsQ = useOrganizationAccounts(token, organizationId);
  const connectionsQ = useConnections(token, organizationId);
  const createStackUrl = useCreateCloudFormationStackUrl(token);
  const updateAccount = useUpdateOrganizationAccount(token);
  const deleteAccount = useDeleteOrganizationAccount(token);
  const createM = useCreateConnection(token);

  const [name, setName] = useState("");
  const [nameShown, setNameShown] = useState(false);
  const [region, setRegion] = useState(() => startRegion(regions, DEFAULT_AWS_REGION));
  const [roleArn, setRoleArn] = useState("");
  const [arnShown, setArnShown] = useState(false);
  const [started, setStarted] = useState<Started | null>(null);
  const [resumeId, setResumeId] = useState<string | null>(null);
  // A linked account chosen to connect, and the connection's name for it.
  const [chosenId, setChosenId] = useState<string | null>(null);
  const [connectionName, setConnectionName] = useState("");
  const [openError, setOpenError] = useState<string | null>(null);
  const [arnRefused, setArnRefused] = useState<string | null>(null);
  const [connectError, setConnectError] = useState<string | null>(null);
  const [linking, setLinking] = useState(false);
  const [removing, setRemoving] = useState<OrganizationAwsAccount | null>(null);
  const [removeError, setRemoveError] = useState<string | null>(null);

  const accounts = accountsInOrder(accountsQ.data ?? []);
  // A link opened here and since removed no longer counts.
  const startedLink = started && (accountsQ.isFetching || accounts.some((a) => a.id === started.account_id)) ? started : null;
  const resumed = startedLink ? null : resumeTarget(accounts, resumeId);
  // The unfinished link the steps work on: the one opened here, or one resumed.
  const active = startedLink ? { id: startedLink.account_id, name: startedLink.name } : resumed ? { id: resumed.id, name: resumed.account_name } : null;
  const chosen = accounts.find((a) => a.id === chosenId && a.status === "complete") ?? null;

  const nameProblem = name.trim() ? awsAccountNameProblem(name) : "Enter a name for this account.";
  const arnProblem = arnRefused ?? roleArnProblem(roleArn);
  const arnReady = !!active && roleArn.trim() !== "" && !roleArnProblem(roleArn);
  const busy = linking || createM.isPending;

  const connect = async (accountId: string, connName: string, fakeLocal: boolean) => {
    setConnectError(null);
    try {
      const made = await createM.mutateAsync({ organizationId, data: { connector: manifest.connector, name: connName, region, linked_id: accountId } });
      onConnected(made);
    } catch (err) {
      setConnectError(connectFailure(err, fakeLocal));
    }
  };

  const openAws = async () => {
    setNameShown(true);
    setOpenError(null);
    if (nameProblem) {
      document.getElementById(id("name"))?.focus();
      return;
    }
    // Opened before the request, so the browser treats it as the click's own window.
    const popup = window.open("", "_blank");
    try {
      const response = await createStackUrl.mutateAsync({ organizationId, accountName: name.trim() });
      setStarted({ ...response, name: name.trim() });
      if (popup) {
        popup.location.href = response.url;
        popup.focus();
      }
    } catch (err) {
      popup?.close();
      // A refused request (400) says why; anything else is worth a retry.
      const reason = err instanceof ApiError && err.status === 400 ? apiErrorMessage(err) : null;
      setOpenError(reason ? `Clutter could not prepare the AWS setup: ${reason}` : "Clutter could not prepare the AWS setup. Try again.");
    }
  };

  // Links the account with the pasted ARN, then connects it.
  const linkAndConnect = async (e: React.FormEvent) => {
    e.preventDefault();
    setArnShown(true);
    if (!active || !arnReady) return;
    setLinking(true);
    setConnectError(null);
    const arn = roleArn.trim();
    try {
      await updateAccount.mutateAsync({ organizationId, accountId: active.id, data: { role_arn: arn } });
    } catch (err) {
      setLinking(false);
      // The backend refuses an ARN whose role was not made by the stack Clutter opened for this link.
      if (err instanceof ApiError && err.status === 400) {
        setArnRefused(`This role does not belong to the stack Clutter opened for ${active.name}. Copy RoleArn from that stack's Outputs tab, or remove this link and start again.`);
      } else {
        setConnectError("Clutter could not link the account. Try again.");
      }
      return;
    }
    // Linked: from here the account is one of the team's linked accounts, so
    // a refused connection is retried from its row without linking again.
    setStarted(null);
    setRoleArn("");
    setArnShown(false);
    setChosenId(active.id);
    setConnectionName(active.name.slice(0, NAME_MAX));
    setLinking(false);
    await connect(active.id, active.name, isFakeLocalRole(arn));
  };

  const remove = async () => {
    if (!removing) return;
    setRemoveError(null);
    try {
      await deleteAccount.mutateAsync({ organizationId, accountId: removing.id });
    } catch {
      setRemoveError("Clutter could not remove the account link. Try again.");
      return;
    }
    toast.success(`Removed the link to ${removing.account_name}`);
    if (started?.account_id === removing.id) setStarted(null);
    if (chosenId === removing.id) setChosenId(null);
    setRoleArn("");
    setArnRefused(null);
    setRemoving(null);
  };

  const askRemove = (a: OrganizationAwsAccount) => {
    setRemoveError(null);
    setRemoving(a);
  };

  if (!entry.has_linked_accounts) {
    return (
      <Shell manifestTitle={manifest.title} look={<CloudMark cloud={cloudLook(manifest)} size={24} />}>
        <Notice tone="error">This Clutter server cannot read linked {manifest.title} accounts, so it cannot connect one.</Notice>
      </Shell>
    );
  }

  return (
    <Shell manifestTitle={manifest.title} look={<CloudMark cloud={cloudLook(manifest)} size={24} />}>
      <ConnectLocalAccount
        token={token}
        organizationId={organizationId}
        connector={manifest.connector}
        cloudName={shortName}
        name={localConnectionName(shortName, (connectionsQ.data ?? []).map((c) => c.name))}
        region={DEFAULT_AWS_REGION}
        onConnected={onConnected}
      />

      {accountsQ.isLoading ? (
        <div role="status" aria-label="Loading your AWS accounts" className="space-y-3">
          <Skeleton className="h-16 rounded-lg" />
          <Skeleton className="h-40 rounded-lg" />
        </div>
      ) : accountsQ.isError ? (
        <Notice tone="error">
          <p>Clutter could not load your AWS account links.</p>
          <Button type="button" size="xs" variant="outline" onClick={() => void accountsQ.refetch()}>
            Retry
          </Button>
        </Notice>
      ) : chosen ? (
        // Connect an account linked earlier, or just now.
        <form
          onSubmit={(e) => {
            e.preventDefault();
            const n = connectionName.trim();
            if (!n) return setConnectError("Give the connection a name.");
            void connect(chosen.id, n, isFakeLocalRole(chosen.role_arn));
          }}
          noValidate
          className="space-y-4 animate-fade-in">
          <div className="bg-muted/40 flex flex-wrap items-center justify-between gap-2 rounded-lg border px-4 py-3">
            <p className="text-sm">
              <span className="text-muted-foreground">Linked account </span>
              <span className="font-medium">{chosen.account_name}</span>
            </p>
            <Button type="button" variant="link" size="sm" className="h-auto px-0" onClick={() => setChosenId(null)}>
              Use another account
            </Button>
          </div>
          <div className={cn("grid gap-4", !narrow && "sm:grid-cols-2")}>
            <Field id={id("conn-name")} label="Connection name" hint="A name your team will recognise.">
              <Input
                id={id("conn-name")}
                value={connectionName}
                onChange={(e) => {
                  setConnectionName(e.target.value);
                  setConnectError(null);
                }}
                maxLength={NAME_MAX}
                autoComplete="off"
                aria-describedby={describedBy(id("conn-name"), true, false)}
              />
            </Field>
            <RegionField id={id("conn-region")} regions={regions} value={region} onChange={setRegion} />
          </div>
          {connectError && <Notice tone="error">{connectError}</Notice>}
          <SubmitButton busy={busy} narrow={narrow} shortName={shortName} />
        </form>
      ) : (
        <>
          {accounts.some((a) => a.id !== active?.id) && (
            <AccountList
              // The link the steps below work on is not listed twice.
              accounts={accounts.filter((a) => a.id !== active?.id)}
              onConnect={(a) => {
                setChosenId(a.id);
                setConnectionName(a.account_name.slice(0, NAME_MAX));
                setConnectError(null);
              }}
              onResume={(a) => {
                setStarted(null);
                setResumeId(a.id);
                setRoleArn("");
                setArnRefused(null);
              }}
              onRemove={askRemove}
            />
          )}

          <form onSubmit={(e) => void linkAndConnect(e)} noValidate>
            <ol className="space-y-6">
              <Step n={1} done={!!active} title={active ? `Name: ${active.name}` : "Name the account and choose a region"}>
                <div className={cn("grid gap-4", !narrow && "sm:grid-cols-2")}>
                  <Field
                    id={id("name")}
                    label="Name"
                    hint={active ? "The connection takes the account's name." : `The account and its connection take this name. ${awsAccountNameHint}`}
                    error={!active && nameShown ? nameProblem : null}>
                    <Input
                      id={id("name")}
                      value={active ? active.name : name}
                      onChange={(e) => {
                        setName(e.target.value);
                        setOpenError(null);
                      }}
                      onBlur={() => name.trim() && setNameShown(true)}
                      disabled={!!active}
                      maxLength={AWS_ACCOUNT_NAME_MAX}
                      placeholder="for example Production"
                      autoComplete="off"
                      aria-invalid={(!active && nameShown && !!nameProblem) || undefined}
                      aria-describedby={describedBy(id("name"), true, !active && nameShown && !!nameProblem)}
                    />
                  </Field>
                  <RegionField id={id("region")} regions={regions} value={region} onChange={setRegion} hint="Where your diagrams build. Each connection has one region." />
                </div>
              </Step>

              <Step n={2} done={!!active} title="Create Clutter's role in AWS">
                {!active ? (
                  <div className="space-y-3">
                    <BeforeYouStart
                      items={AWS_BEFORE_YOU_START}
                      footer={
                        <a
                          href={AWS_SETUP_GUIDE_URL}
                          target="_blank"
                          rel="noreferrer"
                          className="text-link inline-flex items-center gap-1.5 font-medium hover:underline">
                          Open the guide with screenshots
                          <ExternalLink className="size-3.5" aria-hidden="true" />
                        </a>
                      }
                    />
                    <p className="text-muted-foreground text-sm">
                      AWS opens in a new tab with Clutter&apos;s CloudFormation stack filled in. Create the stack and wait until it says CREATE_COMPLETE. AWS
                      opens in Canada (Central); the role works in every region.
                    </p>
                    <Button type="button" onClick={() => void openAws()} disabled={createStackUrl.isPending}>
                      {createStackUrl.isPending ? <Loader2 className="animate-spin" aria-hidden /> : <ExternalLink aria-hidden />}
                      {createStackUrl.isPending ? "Opening AWS…" : "Open AWS"}
                    </Button>
                    {openError && <Notice tone="error">{openError}</Notice>}
                  </div>
                ) : startedLink ? (
                  <div className="flex flex-col gap-2 text-sm sm:flex-row sm:items-center sm:justify-between">
                    <p className="text-muted-foreground">AWS opened in a new tab. Create the stack there, then copy RoleArn from its Outputs tab.</p>
                    <Button type="button" variant="outline" size="sm" onClick={() => window.open(startedLink.url, "_blank", "noopener,noreferrer")}>
                      Open AWS again
                      <ExternalLink aria-hidden />
                    </Button>
                  </div>
                ) : (
                  <Notice tone="info">
                    <p>
                      You started linking <span className="font-medium">{active.name}</span>. Finish the stack in AWS, then paste its role ARN below. Closed AWS
                      before the stack finished? Remove this link and start again.
                    </p>
                    {resumed && (
                      <Button type="button" variant="outline" size="xs" onClick={() => askRemove(resumed)}>
                        <Trash2 aria-hidden />
                        Remove this link
                      </Button>
                    )}
                  </Notice>
                )}
              </Step>

              <Step n={3} done={false} title="Paste the role ARN">
                <div className="space-y-4">
                  <Field
                    id={id("arn")}
                    label="Role ARN"
                    hint={active ? "From the stack's Outputs tab, the value of RoleArn." : "This opens once AWS is open."}
                    error={arnShown || arnRefused ? arnProblem : null}>
                    <Input
                      id={id("arn")}
                      value={roleArn}
                      onChange={(e) => {
                        setRoleArn(e.target.value);
                        setArnRefused(null);
                        setConnectError(null);
                      }}
                      onBlur={() => roleArn.trim() && setArnShown(true)}
                      onPaste={() => setArnShown(true)}
                      disabled={!active || busy}
                      placeholder="arn:aws:iam::123456789012:role/AllowClutterToDeployTerraformRole-…"
                      spellCheck={false}
                      autoComplete="off"
                      className="font-mono text-xs md:text-xs"
                      aria-invalid={((arnShown || !!arnRefused) && !!arnProblem) || undefined}
                      aria-describedby={describedBy(id("arn"), true, (arnShown || !!arnRefused) && !!arnProblem)}
                    />
                  </Field>
                  {connectError && <Notice tone="error">{connectError}</Notice>}
                  <SubmitButton busy={busy} narrow={narrow} shortName={shortName} disabled={!arnReady} linking={linking} />
                </div>
              </Step>
            </ol>
          </form>
        </>
      )}

      {removing && (
        <ConfirmDialog
          title="Remove this AWS link?"
          confirmLabel="Remove link"
          pendingLabel="Removing…"
          isPending={deleteAccount.isPending}
          error={removeError}
          onCancel={() => setRemoving(null)}
          onConfirm={() => void remove()}>
          <p>
            Remove <span className="text-foreground font-medium">{removing.account_name}</span> from this team? To use this AWS account again, you link it
            again.
          </p>
          <p>The role stays in your AWS account until you delete its stack there.</p>
        </ConfirmDialog>
      )}
    </Shell>
  );
}

function Shell({ manifestTitle, look, children }: { manifestTitle: string; look: ReactNode; children: ReactNode }) {
  return (
    <section aria-label={`Connect ${manifestTitle}`} className="space-y-5 animate-fade-in">
      <h3 className="text-foreground flex items-center gap-2.5 text-base font-semibold">
        {look}
        Connect {manifestTitle}
      </h3>
      {children}
    </section>
  );
}

// One numbered step of the link. A finished step shows a tick.
function Step({ n, done, title, children }: { n: number; done: boolean; title: string; children: ReactNode }) {
  return (
    <li className="grid grid-cols-[1.75rem_1fr] gap-x-3">
      <span
        aria-hidden="true"
        className={cn(
          "flex size-7 items-center justify-center rounded-full border text-xs font-semibold transition-colors motion-base",
          done ? "border-primary bg-primary text-primary-foreground" : "bg-background text-muted-foreground",
        )}>
        {done ? <Check className="size-3.5 animate-check" strokeWidth={3} /> : n}
      </span>
      <div className="min-w-0 space-y-3 pt-0.5">
        <p className="text-foreground text-sm font-semibold">
          <span className="sr-only">
            Step {n}
            {done ? ", done" : ""}:{" "}
          </span>
          {title}
        </p>
        {children}
      </div>
    </li>
  );
}

function RegionField({ id, regions, value, onChange, hint }: { id: string; regions: string[]; value: string; onChange: (v: string) => void; hint?: string }) {
  return (
    <Field id={id} label="Region" hint={hint}>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id={id} className="w-full" aria-describedby={describedBy(id, !!hint, false)}>
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
    </Field>
  );
}

function SubmitButton({ busy, narrow, shortName, disabled = false, linking = false }: { busy: boolean; narrow: boolean; shortName: string; disabled?: boolean; linking?: boolean }) {
  return (
    <Button type="submit" disabled={busy || disabled} className={cn(narrow && "w-full")}>
      {busy ? (
        <>
          <Loader2 className="animate-spin" aria-hidden />
          {linking ? "Linking the account…" : `Checking with ${shortName}…`}
        </>
      ) : (
        "Check and connect"
      )}
    </Button>
  );
}

const STATUS: Record<string, { label: string; variant: "success" | "warning" | "neutral" }> = {
  complete: { label: "Linked", variant: "success" },
  incomplete: { label: "Unfinished", variant: "warning" },
  revoked: { label: "Revoked", variant: "neutral" },
};

// The team's AWS account links: connect a linked one, resume an unfinished
// one, or remove either.
function AccountList({
  accounts,
  onConnect,
  onResume,
  onRemove,
}: {
  accounts: OrganizationAwsAccount[];
  onConnect: (a: OrganizationAwsAccount) => void;
  onResume: (a: OrganizationAwsAccount) => void;
  onRemove: (a: OrganizationAwsAccount) => void;
}) {
  return (
    <div className="space-y-2">
      <p className="micro-label">AWS accounts linked to this team</p>
      <ul className="divide-y rounded-lg border">
        {accounts.map((a) => {
          const status = STATUS[a.status] ?? { label: a.status, variant: "neutral" as const };
          return (
            <li key={a.id} className="flex flex-wrap items-center justify-between gap-x-3 gap-y-2 px-3.5 py-2.5">
              <div className="flex min-w-0 flex-wrap items-center gap-2">
                <span className="truncate text-sm font-medium">{a.account_name}</span>
                <Badge variant={status.variant}>{status.label}</Badge>
                {isFakeLocalRole(a.role_arn) && <Badge variant="outline">Fake, local</Badge>}
              </div>
              <div className="flex items-center gap-1">
                {a.status === "complete" && (
                  <Button type="button" size="xs" variant="outline" onClick={() => onConnect(a)}>
                    Connect
                  </Button>
                )}
                {a.status === "incomplete" && (
                  <Button type="button" size="xs" variant="outline" onClick={() => onResume(a)}>
                    Resume
                  </Button>
                )}
                <Button type="button" size="icon-xs" variant="ghost" onClick={() => onRemove(a)} aria-label={`Remove the link to ${a.account_name}`}>
                  <Trash2 aria-hidden />
                </Button>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
