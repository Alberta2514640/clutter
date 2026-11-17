import { Card, CardContent } from "@/components/ui/card";
import type { ProjectRecentActivityItem } from "@/lib/features/logs/types";
import { Activity } from "lucide-react";
import Link from "next/link";

interface ActivitySectionProps {
  activity?: ProjectRecentActivityItem[];
  isLoading?: boolean;
}

export default function ActivitySection({
  activity = [],
  isLoading = false,
}: ActivitySectionProps) {
  const getStatusDotColor = (status: string) => {
    switch (status.toUpperCase()) {
      case "SUCCESS":
        return "bg-green-500";
      case "FAILED":
        return "bg-red-500";
      case "RUNNING":
        return "bg-blue-500 animate-pulse";
      case "QUEUED":
        return "bg-yellow-500";
      default:
        return "bg-gray-500";
    }
  };

  // The user's words for the runner's commands.
  const commandLabel = (command: string) => {
    switch (command.toLowerCase()) {
      case "plan":
        return "Plan";
      case "apply":
        return "Deploy";
      case "destroy":
        return "Destroy";
      default:
        return command;
    }
  };

  const statusLabel = (status: string) => status.charAt(0).toUpperCase() + status.slice(1).toLowerCase();

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div>
      <h2 className="mb-1 flex items-center gap-2 text-2xl font-bold">
        <Activity className="h-6 w-6 text-teal-400" aria-hidden="true" />
        Runs from the older AWS editor
      </h2>
      <p className="mb-4 text-sm text-gray-400">Building-block diagrams show their runs on each diagram&apos;s Deploy tab.</p>

      {isLoading ? (
        <Card className="border-slate-800/50 bg-slate-900/50 backdrop-blur-xl">
          <CardContent className="py-12 text-center text-gray-400">
            Loading activity...
          </CardContent>
        </Card>
      ) : activity.length === 0 ? (
        <Card className="border-slate-800/50 bg-slate-900/50 backdrop-blur-xl">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Activity className="mb-4 h-16 w-16 text-gray-600" />
            <h3 className="mb-2 text-xl font-semibold text-white">No runs yet</h3>
            <p className="text-center text-gray-400">Plan, Deploy and Destroy runs of your older AWS diagrams show here.</p>
          </CardContent>
        </Card>
      ) : (
        <Card className="border-slate-800/50 bg-slate-900/50 backdrop-blur-xl">
          <CardContent className="p-0">
            <div className="divide-y divide-slate-800">
              {activity.map((item) => (
                <Link
                  key={`${item.diagramId}-${item.commandId}`}
                  href={`/projects/${item.projectId}/diagram/${item.diagramId}`}
                  className="block p-4 transition-colors hover:bg-slate-800/30"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <div
                      className={`h-2 w-2 shrink-0 rounded-full ${getStatusDotColor(item.status)}`}
                      aria-hidden="true"
                    />

                    <div className="min-w-0">
                      <p className="truncate font-medium text-white">
                        {item.projectName}
                      </p>
                      <p className="truncate text-sm text-gray-400">
                        {commandLabel(item.command)} • {item.diagramName} • {statusLabel(item.status)}
                      </p>
                      <p className="text-xs text-gray-500">
                        {formatDate(item.createdAt)}
                      </p>
                    </div>
                  </div>
                </Link>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}