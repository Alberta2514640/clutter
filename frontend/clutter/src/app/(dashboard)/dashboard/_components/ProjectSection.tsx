'use client';

import { useRouter } from 'next/navigation';
import Link from 'next/link';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { FolderOpen, Clock, Users, Plus } from 'lucide-react';

interface Project {
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  memberCount?: number;
}

interface ProjectsSectionProps {
  projects: Project[];
}

export default function ProjectsSection({ projects }: ProjectsSectionProps) {
  const router = useRouter();

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
    <div className="mb-8">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <FolderOpen className="w-6 h-6 text-teal-400" />
        Your Projects
      </h2>
      
      {projects.length === 0 ? (
        <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
          <CardContent className="flex flex-col items-center justify-center py-12">
            <FolderOpen className="w-16 h-16 text-gray-600 mb-4" />
            <h3 className="text-xl font-semibold text-white mb-2">No projects yet</h3>
            <p className="text-gray-400 text-center mb-6">
              Create your first project to start building infrastructure
            </p>
            <Button 
              onClick={() => router.push('/projects/new')}
              className="bg-teal-500 hover:bg-teal-600"
            >
              <Plus className="w-4 h-4 mr-2" />
              Create Project
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects.map((project) => (
            <Link key={project.projectId} href={`/projects/${project.projectId}`}>
              <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50 hover:border-teal-500/50 transition-all cursor-pointer group">
                <CardHeader>
                  <CardTitle className="text-white group-hover:text-teal-400 transition-colors">
                    {project.name}
                  </CardTitle>
                  <CardDescription className="text-gray-400 line-clamp-2">
                    {project.description || 'No description'}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="flex items-center justify-between text-sm text-gray-500">
                    <span className="flex items-center gap-1">
                      <Clock className="w-4 h-4" />
                      {formatDate(project.updatedAt)}
                    </span>
                    {project.memberCount && (
                      <span className="flex items-center gap-1">
                        <Users className="w-4 h-4" />
                        {project.memberCount}
                      </span>
                    )}
                  </div>
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
