"use client";
import { Card, CardContent } from "@/components/ui/card";
import { useUserState } from "@/lib/stores/userStore";
import { FileText, LayoutTemplate } from "lucide-react";

export default function ProjectOverview() {
  const { user } = useUserState();
  
  // Get first name or fallback to "there"
  const firstName = user?.displayName?.split(' ')[0] || "there";

  return (
    <div className="max-w-5xl mx-auto px-6 py-12">
      <div className="text-center mb-12">
        <div className="inline-flex items-center justify-center w-16 h-16 mb-4 rounded-2xl bg-gradient-to-br from-teal-500 to-blue-600">
          <span className="text-3xl">👋</span>
        </div>
        <h2 className="text-3xl font-bold text-white mb-2">
          Welcome {firstName}!
        </h2>
        <p className="text-gray-400 text-lg">Create your first workflow</p>
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all cursor-pointer group">
          <CardContent className="p-8 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
              <FileText className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              Start from scratch
            </h3>
          </CardContent>
        </Card>
        <Card className="bg-slate-800/50 border-slate-700/50 hover:border-teal-500/50 transition-all cursor-pointer group">
          <CardContent className="p-8 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 mb-6 rounded-xl bg-slate-700/50 group-hover:bg-teal-500/20 transition-colors">
              <LayoutTemplate className="w-8 h-8 text-gray-400 group-hover:text-teal-400 transition-colors" />
            </div>
            <h3 className="text-lg font-semibold text-white mb-2">
              Start with a template
            </h3>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}