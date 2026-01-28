import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Activity, FolderOpen, Users } from 'lucide-react';

interface StatsCardsProps {
  projectCount: number;
  diagramCount: number;
  memberCount: number;
}

export default function StatsCards({ projectCount, diagramCount, memberCount }: StatsCardsProps) {
  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-gray-400">
            Total Projects
          </CardTitle>
          <FolderOpen className="w-4 h-4 text-teal-400" />
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold text-white">{projectCount}</div>
        </CardContent>
      </Card>

      <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-gray-400">
            Total Diagrams
          </CardTitle>
          <Activity className="w-4 h-4 text-teal-400" />
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold text-white">{diagramCount}</div>
        </CardContent>
      </Card>

      <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-gray-400">
            Team Members
          </CardTitle>
          <Users className="w-4 h-4 text-teal-400" />
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold text-white">{memberCount}</div>
        </CardContent>
      </Card>
    </div>
  );
}
