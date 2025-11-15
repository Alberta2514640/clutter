import Link from 'next/link';
import { Card, CardContent } from '@/components/ui/card';
import { Activity } from 'lucide-react';

interface Run {
  runId: string;
  projectId: string;
  projectName: string;
  workspaceId: string;
  action: 'plan' | 'apply';
  status: 'QUEUED' | 'RUNNING' | 'SUCCESS' | 'FAILED';
  startedAt: string;
  endedAt?: string;
}

interface ActivitySectionProps {
  runs: Run[];
}

export default function ActivitySection({ runs }: ActivitySectionProps) {
  const getStatusColor = (status: Run['status']) => {
    switch (status) {
      case 'SUCCESS': return 'text-green-500';
      case 'FAILED': return 'text-red-500';
      case 'RUNNING': return 'text-blue-500';
      case 'QUEUED': return 'text-yellow-500';
      default: return 'text-gray-500';
    }
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };

  return (
    <div>
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <Activity className="w-6 h-6 text-teal-400" />
        Recent Activity
      </h2>
      
      {runs.length === 0 ? (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Activity className="w-16 h-16 text-gray-600 mb-4" />
            <h3 className="text-xl font-semibold text-white mb-2">No activity yet</h3>
            <p className="text-gray-400 text-center">
              Deploy your first workspace to see activity here
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent className="p-0">
            <div className="divide-y divide-slate-800">
              {runs.map((run) => (
                <Link 
                  key={run.runId} 
                  href={`/projects/${run.projectId}/runs/${run.runId}`}
                  className="block p-4 hover:bg-slate-800/30 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <div className={`w-2 h-2 rounded-full ${
                        run.status === 'SUCCESS' ? 'bg-green-500' :
                        run.status === 'FAILED' ? 'bg-red-500' :
                        run.status === 'RUNNING' ? 'bg-blue-500 animate-pulse' :
                        'bg-yellow-500'
                      }`} />
                      <div>
                        <p className="text-white font-medium">{run.projectName}</p>
                        <p className="text-sm text-gray-400">
                          {run.action.toUpperCase()} • {run.workspaceId}
                        </p>
                      </div>
                    </div>
                    <div className="text-right">
                      <p className={`text-sm font-medium ${getStatusColor(run.status)}`}>
                        {run.status}
                      </p>
                      <p className="text-xs text-gray-500">
                        {formatDate(run.startedAt)}
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