"use client";

import PageHeader from "@/components/common/PageHeader";
import PageSkeleton from "@/components/common/PageSkeleton";
import ThemeToggle from "@/components/common/ThemeToggle";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { toast } from "@/components/ui/sonner";
import { effectiveExperience, NOT_SURE_CLOUD, usePreferencesStore, type Experience } from "@/lib/features/preferences/uiStore";
import { useLogout, useMe } from "@/lib/features/user/hooks";
import { useIsClient } from "@/lib/features/user/useIsClient";
import { LogOut } from "lucide-react";
import { useRouter } from "next/navigation";
import type { ReactNode } from "react";

// You: the Google profile Clutter knows (read-only), this browser's
// preferences, and signing out. Clutter has no account settings of its own;
// deleting an account is not offered (A has no way to do it).

const EXPERIENCE_OPTIONS: { value: Experience; label: string }[] = [
  { value: "new", label: "New to cloud setups" },
  { value: "console", label: "Used a cloud console, not code" },
  { value: "terraform", label: "Writes Terraform or similar code" },
];

const CLOUD_OPTIONS = [
  { value: "aws", label: "AWS" },
  { value: "general-openstack", label: "OpenStack" },
  { value: NOT_SURE_CLOUD, label: "Not sure yet" },
];

function Section({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <section className="surface-card">
      <div className="border-b p-5 md:px-6">
        <h2 className="text-foreground text-base font-semibold">{title}</h2>
        <p className="text-muted-foreground mt-1 text-sm">{description}</p>
      </div>
      <div className="p-5 md:p-6">{children}</div>
    </section>
  );
}

function Row({ label, htmlFor, hint, children }: { label: string; htmlFor?: string; hint: string; children: ReactNode }) {
  return (
    <div className="grid gap-3 py-4 first:pt-0 last:pb-0 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-center sm:gap-6">
      <div className="min-w-0">
        {htmlFor ? <Label htmlFor={htmlFor}>{label}</Label> : <p className="text-sm font-medium">{label}</p>}
        <p className="text-muted-foreground mt-1 text-sm">{hint}</p>
      </div>
      <div className="min-w-0">{children}</div>
    </div>
  );
}

const initialsOf = (name: string) =>
  name
    .trim()
    .split(/\s+/)
    .slice(0, 2)
    .map((p) => p[0])
    .join("")
    .toUpperCase() || "?";

export default function AccountPage() {
  const router = useRouter();
  const meQ = useMe();
  const logout = useLogout();
  const isClient = useIsClient();
  const experience = usePreferencesStore((s) => s.experience);
  const firstCloud = usePreferencesStore((s) => s.firstCloud);
  const setExperience = usePreferencesStore((s) => s.setExperience);
  const setFirstCloud = usePreferencesStore((s) => s.setFirstCloud);

  if (meQ.isLoading || !isClient) return <PageSkeleton label="Loading your account…" cards={2} />;
  const me = meQ.data;
  if (!me) return null;

  const signOut = () => {
    logout();
    toast.success("You are signed out.");
    router.push("/login");
    router.refresh();
  };

  const cloudValue = firstCloud && CLOUD_OPTIONS.some((o) => o.value === firstCloud) ? firstCloud : NOT_SURE_CLOUD;

  return (
    <div className="max-w-3xl">
      <PageHeader eyebrow="You" title="Account and preferences" description="Who you are signed in as, and how Clutter behaves in this browser." />

      <div className="space-y-8">
        <Section title="Account" description="From your Google account. Change your name or photo in Google; Clutter shows them as Google sends them.">
          <div className="flex items-center gap-4">
            <Avatar className="size-14">
              {me.pictureUrl && <AvatarImage src={me.pictureUrl} alt="" referrerPolicy="no-referrer" />}
              <AvatarFallback className="bg-primary/10 text-link font-semibold">{initialsOf(me.displayName || me.email)}</AvatarFallback>
            </Avatar>
            <dl className="min-w-0">
              <dt className="sr-only">Name</dt>
              <dd className="text-foreground truncate text-base font-semibold">{me.displayName}</dd>
              <dt className="sr-only">Email</dt>
              <dd className="text-muted-foreground truncate text-sm">{me.email}</dd>
            </dl>
          </div>
        </Section>

        <Section title="Preferences" description="Kept in this browser only. They change defaults, never your diagrams.">
          <div className="divide-y">
            <Row label="Theme" hint="Light, dark, or the same as your device.">
              <ThemeToggle />
            </Row>
            <Row
              label="Experience"
              htmlFor="pref-experience"
              hint="New and console users get the first-deploy checklist open, and new users get a one-line hint on each block. Terraform users get the code preview open.">
              <Select
                value={effectiveExperience(experience)}
                onValueChange={(v) => {
                  const next = EXPERIENCE_OPTIONS.find((o) => o.value === v);
                  if (!next) return;
                  setExperience(next.value);
                  toast.success("Preference saved");
                }}>
                <SelectTrigger id="pref-experience" className="w-full sm:w-72">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {EXPERIENCE_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Row>
            <Row label="First cloud" htmlFor="pref-cloud" hint="Shown first in examples, and the cloud the checklist's Connect step opens.">
              <Select
                value={cloudValue}
                onValueChange={(v) => {
                  setFirstCloud(v);
                  toast.success("Preference saved");
                }}>
                <SelectTrigger id="pref-cloud" className="w-full sm:w-72">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {CLOUD_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Row>
          </div>
        </Section>

        <Section title="Sign out" description="Signs you out in this browser. Your teams and diagrams stay as they are.">
          <Button type="button" variant="outline" onClick={signOut}>
            <LogOut aria-hidden="true" />
            Sign out
          </Button>
        </Section>
      </div>
    </div>
  );
}
