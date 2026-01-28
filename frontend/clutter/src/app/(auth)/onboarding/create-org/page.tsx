"use client";

import { AlertCircle, ArrowRight, Building2, ShieldCheck, Sparkles, Users } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { useCreateOrganization, useOrganizations } from "@/lib/features/organization/hooks";
import { useMe } from "@/lib/features/user/hooks";

export default function CreateOrgPage() {
  const router = useRouter();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgsQ = useOrganizations(token);
  const createOrg = useCreateOrganization(token);

  const [organizationName, setOrganizationName] = useState("");
  const [description, setDescription] = useState("");

  const hasOrg = useMemo(() => (orgsQ.data ?? []).length > 0, [orgsQ.data]);

  useEffect(() => {
    if (meQ.isLoading) return;

    // not signed in → login
    if (!token) {
      router.replace("/login");
      return;
    }

    // signed in AND already has org → dashboard
    if (!orgsQ.isLoading && !orgsQ.isError && hasOrg) {
      router.replace("/dashboard");
    }
  }, [meQ.isLoading, token, hasOrg, orgsQ.isLoading, orgsQ.isError, router]);
  
  const canSubmit = organizationName.trim().length >= 2 && !!token && !createOrg.isPending;

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!canSubmit) return;

    await createOrg.mutateAsync({
      organizationName: organizationName.trim(),
      description: description.trim() || "", // backend can store "" or you can allow null if API supports it
    });

    router.replace("/dashboard");
  };

  // Auth not ready / not signed in
  if (!meQ.isLoading && !token) {
    router.replace("/login");
    return null;
  }

  return (
    <div className="min-h-screen px-6 py-12 flex items-center justify-center bg-gradient-to-br from-black via-slate-900 to-teal-900">
      <Card className="max-w-4xl w-full bg-slate-900/50 backdrop-blur-xl border border-slate-800/50 shadow-2xl">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between gap-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-teal-500/10 rounded-lg border border-teal-500/20">
                <Building2 className="w-7 h-7 text-teal-400" />
              </div>

              <div>
                <CardTitle className="text-2xl text-white flex items-center gap-2">
                  Let’s start by creating your organization <Sparkles className="w-5 h-5 text-teal-400" />
                </CardTitle>
                <CardDescription className="text-gray-400 mt-1">
                  This is where you’ll create projects, diagrams, workspaces, and runs. You can invite teammates later.
                </CardDescription>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent className="grid grid-cols-1 lg:grid-cols-2 gap-8 pt-2 items-start">
          {/* LEFT: Form */}
          <div className="space-y-6 pb-8 lg:pb-0 relative z-10">
            {(createOrg.isError || meQ.isError || orgsQ.isError) && (
              <Alert className="bg-red-500/10 border-red-500/30">
                <AlertCircle className="h-4 w-4 text-red-300" />
                <AlertTitle className="text-white">Could not create organization</AlertTitle>
                <AlertDescription className="text-red-200/80">
                  {createOrg.isError && String(createOrg.error)}
                  {meQ.isError && `Auth error: ${String(meQ.error)}`}
                  {orgsQ.isError && `Org lookup error: ${String(orgsQ.error)}`}
                </AlertDescription>
              </Alert>
            )}

            <form onSubmit={onSubmit} className="space-y-5">
              <div className="space-y-2">
                <Label className="text-gray-200">Organization name</Label>
                <Input
                  value={organizationName}
                  onChange={(e) => setOrganizationName(e.target.value)}
                  placeholder="e.g., Clutter Labs"
                  className="bg-slate-900/40 border-slate-700 text-white placeholder:text-gray-500 focus-visible:ring-teal-500"
                  autoFocus
                />
                <p className="text-xs text-gray-500">
                  Stored in PostgreSQL and shown to members in the sidebar and invitations.
                </p>
              </div>

              <div className="space-y-2">
                <Label className="text-gray-200">Description (optional)</Label>
                <Input
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="e.g., Infrastructure diagrams for my team"
                  className="bg-slate-900/40 border-slate-700 text-white placeholder:text-gray-500 focus-visible:ring-teal-500"
                />
                <p className="text-xs text-gray-500">Helps teammates understand what this org is for.</p>
              </div>

              <div className="flex flex-col sm:flex-row gap-3 pt-2">
                <Button
                  type="submit"
                  className="w-full bg-teal-500 hover:bg-teal-600 text-slate-950 font-semibold"
                  disabled={!canSubmit}
                >
                  {createOrg.isPending ? "Creating..." : "Create organization"}
                  <ArrowRight className="w-4 h-4 ml-2" />
                </Button>
              </div>

              <div className="text-xs text-gray-500">
                By creating an org, you will be set as the <span className="text-gray-300">Owner</span>. You can manage roles in{" "}
                <span className="text-gray-300">Settings</span>.
              </div>
            </form>
          </div>

          {/* RIGHT: Details */}
          <div className="space-y-4 lg:sticky lg:top-6">
            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <div className="flex items-center gap-3 mb-2">
                <Users className="w-5 h-5 text-teal-400" />
                <h3 className="text-white font-semibold">Collaboration-ready</h3>
              </div>
              <p className="text-sm text-gray-400">
                Invite members, assign roles (owner/editor/viewer), and collaborate on diagrams with presence + locks.
              </p>
            </div>

            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <div className="flex items-center gap-3 mb-2">
                <ShieldCheck className="w-5 h-5 text-teal-400" />
                <h3 className="text-white font-semibold">Project isolation</h3>
              </div>
              <p className="text-sm text-gray-400">
                Your org is the security boundary. Projects, canvases, nodes, and runs stay scoped to your org.
              </p>
            </div>

            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <h3 className="text-white font-semibold mb-2">What happens next?</h3>
              <ol className="text-sm text-gray-400 space-y-2 list-decimal list-inside">
                <li>Create your first project</li>
                <li>Build a diagram (Lambda / EC2 / DynamoDB / API GW)</li>
                <li>Export Terraform or run Plan/Apply</li>
              </ol>
            </div>

            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <p className="text-xs text-gray-500">Tip: If you’re joining a team, you might already have an invite.</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}