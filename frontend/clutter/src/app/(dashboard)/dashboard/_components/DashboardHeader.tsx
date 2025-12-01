'use client';

import { Button } from '@/components/ui/button';
import { Plus } from 'lucide-react';
import { useCreateProject } from '@/lib/hooks/mutations/useCreateProject';

interface DashboardHeaderProps {
  organizationId: string;
  tenantName?: string;
}

export default function DashboardHeader({ organizationId, tenantName }: DashboardHeaderProps) {
  const { mutate: createProject, isPending } = useCreateProject(organizationId);

  const handleCreateProject = () => {
    createProject({
      name: 'MyProject',
      description: 'Project description',
    });
  };

  return (
    <div className="flex items-center justify-between mb-8">
      <div>
        <h1 className="text-4xl font-bold mb-2">Dashboard</h1>
        <p className="text-gray-400">{tenantName || 'Your Organization'}</p>
      </div>
      <Button
        onClick={handleCreateProject}
        disabled={isPending}
        className="bg-teal-500 hover:bg-teal-600 flex items-center gap-2"
      >
        <Plus className="w-4 h-4" />
        {isPending ? 'Creating...' : 'New Project'}
      </Button>
    </div>
  );
}