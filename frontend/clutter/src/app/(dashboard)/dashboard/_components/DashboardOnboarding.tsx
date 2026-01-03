"use client";

import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";

export default function DashboardOnboarding() {
  const router = useRouter();

  return (
    <div className="flex items-center justify-center min-h-[calc(100vh-20rem)] px-6">
      <Card className="max-w-2xl w-full bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
        <CardHeader>
          <div className="flex items-center gap-3 mb-4">
            <div className="p-3 bg-teal-500/10 rounded-lg">
              <AlertCircle className="w-8 h-8 text-teal-400" />
            </div>
            <div>
              <CardTitle className="text-2xl text-white">Welcome to Clutter!</CardTitle>
              <CardDescription className="text-gray-400">Let&apos;s set up your organization</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <Alert className="bg-slate-800/50 border-slate-700">
            <AlertCircle className="h-4 w-4 text-teal-400" />
            <AlertTitle className="text-white">No Organization Found</AlertTitle>
            <AlertDescription className="text-gray-400">You need to create or join an organization before you can start creating projects.</AlertDescription>
          </Alert>

          <div className="space-y-4">
            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <h3 className="text-white font-semibold mb-2">Create a New Organization</h3>
              <p className="text-sm text-gray-400 mb-4">Start fresh with your own organization. You will be the owner and can invite team members later.</p>
              <Button onClick={() => router.push("/onboarding/create-tenant")} className="w-full bg-teal-500 hover:bg-teal-600">
                Create Organization
              </Button>
            </div>

            <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
              <h3 className="text-white font-semibold mb-2">Join an Existing Organization</h3>
              <p className="text-sm text-gray-400 mb-4">Have an invitation code? Join your team&apos;s organization here.</p>
              <Button onClick={() => router.push("/onboarding/join-tenant")} variant="secondary" className="w-full bg-slate-600 hover:bg-slate-700   text-white transition-colors">
                Join Organization
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
