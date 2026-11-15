"use client";

import { AlertCircle, ArrowRight, Building2, Sparkles } from "lucide-react";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useState } from "react";

import FirstSteps from "@/components/common/FirstSteps";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

import { useCreateOrganization, useOrganizations } from "@/lib/features/organization/hooks";
import {
  organizationDescriptionHint,
  organizationDescriptionProblem,
  organizationNameCreateHint,
  organizationNamePlaceholder,
  organizationNameProblem,
} from "@/lib/features/organization/rules";
import { useMe } from "@/lib/features/user/hooks";
import { isConflict } from "@/lib/features/user/session";

// The first organization, right after the first sign-in. A user who already has
// one goes to the dashboard; a further organization is made from the sidebar's
// "New organization" (settings/organization/create-org).
export default function CreateOrgPage() {
  const router = useRouter();

  const meQ = useMe();
  const token = meQ.data?.token ?? null;

  const orgsQ = useOrganizations(token);
  const createOrg = useCreateOrganization(token);

  const [organizationName, setOrganizationName] = useState("");
  const [description, setDescription] = useState("");

  const hasOrg = useMemo(() => (orgsQ.data ?? []).length > 0, [orgsQ.data]);
  const nameProblem = organizationNameProblem(organizationName);
  const descriptionProblem = organizationDescriptionProblem(description);

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

  const canSubmit = !nameProblem && !descriptionProblem && !!token && !createOrg.isPending;

  const onSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!canSubmit) return;

    const trimmedDescription = description.trim();
    try {
      await createOrg.mutateAsync({
        organizationName: organizationName.trim(),
        ...(trimmedDescription ? { description: trimmedDescription } : {}),
      });
      router.replace("/dashboard");
    } catch {
      // Shown from createOrg.error below.
    }
  };

  if (!meQ.isLoading && !token) return null;

  const createErrorText = isConflict(createOrg.error)
    ? "You already have an organization with that name. Choose another name."
    : "Clutter could not create the organization. Try again.";

  return (
    <div className="min-h-screen px-6 py-12 flex items-center justify-center bg-gradient-to-br from-black via-slate-900 to-teal-900">
      <Card className="max-w-4xl w-full bg-slate-900/50 backdrop-blur-xl border border-slate-800/50 shadow-2xl">
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between gap-6">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-teal-500/10 rounded-lg border border-teal-500/20">
                <Building2 className="w-7 h-7 text-teal-400" aria-hidden="true" />
              </div>

              <div>
                <CardTitle className="text-2xl text-white flex items-center gap-2">
                  Create your organization <Sparkles className="w-5 h-5 text-teal-400" aria-hidden="true" />
                </CardTitle>
                <CardDescription className="text-gray-400 mt-1">
                  An organization holds your projects, your diagrams and the cloud accounts you link.
                </CardDescription>
              </div>
            </div>
          </div>
        </CardHeader>

        <CardContent className="grid grid-cols-1 lg:grid-cols-2 gap-8 pt-2 items-start">
          {/* LEFT: Form */}
          <div className="space-y-6 pb-8 lg:pb-0 relative z-10">
            {createOrg.isError && (
              <Alert className="bg-red-500/10 border-red-500/30">
                <AlertCircle className="h-4 w-4 text-red-300" />
                <AlertTitle className="text-white">Could not create the organization</AlertTitle>
                <AlertDescription className="text-red-200/80">{createErrorText}</AlertDescription>
              </Alert>
            )}

            {orgsQ.isError && (
              <Alert className="bg-red-500/10 border-red-500/30">
                <AlertCircle className="h-4 w-4 text-red-300" />
                <AlertTitle className="text-white">Could not check your organizations</AlertTitle>
                <AlertDescription className="text-red-200/80">Check your connection, then reload the page.</AlertDescription>
              </Alert>
            )}

            <form onSubmit={onSubmit} className="space-y-5">
              <div className="space-y-2">
                <Label htmlFor="onboarding-org-name" className="text-gray-200">
                  Organization name
                </Label>
                <Input
                  id="onboarding-org-name"
                  value={organizationName}
                  onChange={(e) => setOrganizationName(e.target.value)}
                  placeholder={organizationNamePlaceholder}
                  aria-describedby="onboarding-org-name-hint"
                  className="bg-slate-900/40 border-slate-700 text-white placeholder:text-gray-500 focus-visible:ring-teal-500"
                  autoFocus
                />
                <p id="onboarding-org-name-hint" className="text-xs text-gray-500">
                  {organizationNameCreateHint}
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="onboarding-org-description" className="text-gray-200">
                  Description (optional)
                </Label>
                <Input
                  id="onboarding-org-description"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                  placeholder="e.g., Infrastructure diagrams for my team"
                  aria-describedby="onboarding-org-description-hint"
                  aria-invalid={!!descriptionProblem}
                  className="bg-slate-900/40 border-slate-700 text-white placeholder:text-gray-500 focus-visible:ring-teal-500"
                />
                <p id="onboarding-org-description-hint" className={descriptionProblem ? "text-xs text-red-300" : "text-xs text-gray-500"}>
                  {descriptionProblem ?? organizationDescriptionHint}
                </p>
              </div>

              <div className="flex flex-col sm:flex-row gap-3 pt-2">
                <Button type="submit" className="w-full bg-teal-500 hover:bg-teal-600 text-slate-950 font-semibold" disabled={!canSubmit}>
                  {createOrg.isPending ? "Creating..." : "Create organization"}
                  <ArrowRight className="w-4 h-4 ml-2" aria-hidden="true" />
                </Button>
              </div>
            </form>
          </div>

          {/* RIGHT: Details */}
          <div className="space-y-4 lg:sticky lg:top-6">
            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <h2 className="text-white font-semibold mb-3">What happens next?</h2>
              <FirstSteps withLinks={false} />
            </div>

            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <p className="text-sm text-gray-400">You can create more organizations later from the sidebar, for example one per team or client.</p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
