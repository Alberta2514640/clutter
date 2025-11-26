"use client";

import Navbar from "@/components/common/Navbar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useRouter } from "next/navigation";
import { useState, FormEvent, ChangeEvent } from "react";
import { Loader2, Building2, ShieldCheck } from "lucide-react";

export default function CreateDefaultOrgPage() {
  const router = useRouter();
  const [orgName, setOrgName] = useState("");
  const [orgSlug, setOrgSlug] = useState("");
  const [description, setDescription] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError(null);

    if (!orgName.trim()) {
      setError("Please enter an organization name.");
      return;
    }

    setIsSubmitting(true);

    try {
      const res = await fetch("/api/onboarding/create-default-org", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: orgName.trim(),
          slug: orgSlug.trim() || undefined,
          description: description.trim() || undefined,
        }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body?.message || "Failed to create organization.");
      }

      // After org is created, go to dashboard (or projects page)
      router.push("/dashboard");
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Something went wrong. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-black text-white flex flex-col pt-20 relative overflow-hidden">
      {/* Fixed frosted navbar */}
      <Navbar showLogin={false} />

      {/* Background theme */}
      <div className="pointer-events-none absolute inset-0 -z-10 overflow-hidden">
        {/* Base gradient */}
        <div className="absolute inset-0 bg-gradient-to-br from-cyan-500/[0.05] via-transparent to-indigo-600/[0.08] blur-3xl" />
        {/* Orbs */}
        <div className="absolute top-[-10%] right-[-20%] w-[600px] h-[600px] bg-cyan-400/20 blur-[160px] rounded-full" />
        <div className="absolute bottom-[-10%] left-[-20%] w-[650px] h-[650px] bg-indigo-500/25 blur-[180px] rounded-full" />
        {/* Vignette */}
        <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-black/50" />
      </div>

      <main className="flex flex-1 items-center justify-center px-6 py-16">
        <div className="w-full max-w-3xl grid gap-10 md:grid-cols-[1.3fr,1fr] items-start">
          {/* Left: copy / explanation */}
          <div className="space-y-6">
            <p className="text-xs font-semibold tracking-[0.25em] text-cyan-300 uppercase">Onboarding</p>
            <h1 className="text-3xl md:text-4xl font-semibold tracking-tight">Create your default organization.</h1>
            <p className="text-sm md:text-base text-slate-300/80 leading-relaxed">
              Organizations (tenants) are the top-level boundary in Clutter. All projects, canvases, workspaces, and runs will live inside this org. You can invite teammates later and add more environments as you grow.
            </p>

            <div className="grid gap-3 text-xs text-slate-300/90">
              <div className="flex gap-3 items-start">
                <div className="mt-0.5 rounded-md bg-cyan-500/10 border border-cyan-500/40 p-1.5">
                  <Building2 className="h-4 w-4 text-cyan-300" />
                </div>
                <div>
                  <p className="font-medium mb-0.5">Tenant boundary</p>
                  <p className="text-slate-400">Keep projects, AWS accounts, and IaC runs logically grouped under a single organization.</p>
                </div>
              </div>

              <div className="flex gap-3 items-start">
                <div className="mt-0.5 rounded-md bg-emerald-500/10 border border-emerald-500/40 p-1.5">
                  <ShieldCheck className="h-4 w-4 text-emerald-300" />
                </div>
                <div>
                  <p className="font-medium mb-0.5">Access &amp; roles</p>
                  <p className="text-slate-400">Later, assign owners, editors, and viewers to projects inside this org to control who can edit diagrams and run deployments.</p>
                </div>
              </div>
            </div>
          </div>

          {/* Right: form card */}
          <div className="w-full">
            <div className="bg-slate-950/80 backdrop-blur-xl rounded-2xl border border-slate-800/80 p-6 md:p-7 shadow-[0_18px_60px_rgba(0,0,0,0.85)]">
              <h2 className="text-lg font-semibold mb-1">Default organization</h2>
              <p className="text-xs text-slate-400 mb-6">You can change these details later from Settings.</p>

              <form onSubmit={handleSubmit} className="space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="orgName" className="text-xs text-slate-200">
                    Organization name
                  </Label>
                  <Input id="orgName" value={orgName} onChange={(e: ChangeEvent<HTMLInputElement>) => setOrgName(e.target.value)} placeholder="e.g., Kokonut Cloud Studio" className="bg-slate-900/70 border-slate-700 text-sm" />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="orgSlug" className="text-xs text-slate-200">
                    Organization slug
                    <span className="text-slate-500"> (optional)</span>
                  </Label>
                  <Input
                    id="orgSlug"
                    value={orgSlug}
                    onChange={(e: ChangeEvent<HTMLInputElement>) => setOrgSlug(e.target.value.toLowerCase().replace(/\s+/g, "-"))}
                    placeholder="e.g., kokonut-cloud"
                    className="bg-slate-900/70 border-slate-700 text-sm"
                  />
                  <p className="text-[11px] text-slate-500">Used in URLs and internal references. Lowercase, dashes only.</p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description" className="text-xs text-slate-200">
                    Description
                    <span className="text-slate-500"> (optional)</span>
                  </Label>
                  <textarea
                    id="description"
                    value={description}
                    onChange={(e) => setDescription(e.target.value)}
                    rows={3}
                    placeholder="Short description of what this org will manage (teams, environments, etc.)."
                    className="w-full resize-none rounded-md border border-slate-700 bg-slate-900/70 px-3 py-2 text-sm text-slate-100 placeholder:text-slate-500 focus:outline-none focus:ring-1 focus:ring-cyan-500/60"
                  />
                </div>

                {error && <p className="text-xs text-rose-400 bg-rose-950/40 border border-rose-800/60 rounded-md px-3 py-2">{error}</p>}

                <div className="flex flex-col sm:flex-row gap-3 pt-2">
                  <Button type="submit" className="flex-1 bg-cyan-500 hover:bg-cyan-400 text-black font-medium flex items-center justify-center gap-2" disabled={isSubmitting}>
                    {isSubmitting && <Loader2 className="h-4 w-4 animate-spin" />}
                    {isSubmitting ? "Creating..." : "Create organization"}
                  </Button>

                  <Button type="button" variant="outline" className="border-slate-700 text-slate-200 hover:bg-slate-900" onClick={() => router.push("/dashboard")}>
                    Skip for now
                  </Button>
                </div>
              </form>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
