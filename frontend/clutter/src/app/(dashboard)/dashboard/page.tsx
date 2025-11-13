// 'use client';

// import { useEffect, useState } from 'react';
// import { useRouter } from 'next/navigation';
// import Navbar from '@/components/common/Navbar';
// import { Button } from '@/components/ui/button';
// import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
// import { Skeleton } from '@/components/ui/skeleton';
// import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
// import { Plus, FolderOpen, Activity, Clock, Users, AlertCircle } from 'lucide-react';
// import Link from 'next/link';
// import { apiClient } from '@/lib/api-client';

// interface Project {
//   projectId: string;
//   name: string;
//   description: string;
//   createdAt: string;
//   updatedAt: string;
//   memberCount?: number;
// }

// interface Run {
//   runId: string;
//   projectId: string;
//   projectName: string;
//   workspaceId: string;
//   action: 'plan' | 'apply';
//   status: 'QUEUED' | 'RUNNING' | 'SUCCESS' | 'FAILED';
//   startedAt: string;
//   endedAt?: string;
// }

// interface UserData {
//   userId: string;
//   tenantId: string | null;
//   email: string;
//   displayName: string;
//   tenant?: {
//     tenantId: string;
//     name: string;
//   };
// }

// export default function DashboardPage() {
//   const router = useRouter();
//   const [userData, setUserData] = useState<UserData | null>(null);
//   const [projects, setProjects] = useState<Project[]>([]);
//   const [recentRuns, setRecentRuns] = useState<Run[]>([]);
//   const [loading, setLoading] = useState(true);
//   const [error, setError] = useState<string | null>(null);

//   useEffect(() => {
//     fetchDashboardData();
//   }, []);

//   const fetchDashboardData = async () => {
//     try {
//       setLoading(true);
//       setError(null);

//       // Fetch user data via Lambda
//       const user = await apiClient.getUserProfile();
//       setUserData(user);

//       // Only fetch projects if user has a tenant
//       if (user.tenantId) {
//         const [projectsData, runsData] = await Promise.all([
//           apiClient.getProjects(),
//           apiClient.getRecentRuns()
//         ]);

//         setProjects(projectsData.projects || []);
//         setRecentRuns(runsData.runs || []);
//       }
//     } catch (err) {
//       setError(err instanceof Error ? err.message : 'An error occurred');
//     } finally {
//       setLoading(false);
//     }
//   };

//   const getStatusColor = (status: Run['status']) => {
//     switch (status) {
//       case 'SUCCESS': return 'text-green-500';
//       case 'FAILED': return 'text-red-500';
//       case 'RUNNING': return 'text-blue-500';
//       case 'QUEUED': return 'text-yellow-500';
//       default: return 'text-gray-500';
//     }
//   };

//   const formatDate = (dateString: string) => {
//     return new Date(dateString).toLocaleDateString('en-US', {
//       month: 'short',
//       day: 'numeric',
//       year: 'numeric',
//       hour: '2-digit',
//       minute: '2-digit'
//     });
//   };

//   if (loading) {
//     return (
//       <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900">
//         <Navbar />
//         <main className="max-w-7xl mx-auto px-6 py-24">
//           <div className="space-y-8">
//             <Skeleton className="h-12 w-64 bg-slate-800" />
//             <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
//               {[1, 2, 3].map((i) => (
//                 <Skeleton key={i} className="h-32 bg-slate-800" />
//               ))}
//             </div>
//           </div>
//         </main>
//       </div>
//     );
//   }

//   // No tenant - show onboarding
//   if (userData && !userData.tenantId) {
//     return (
//       <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900">
//         <Navbar />
//         <main className="flex items-center justify-center min-h-screen px-6 py-24">
//           <Card className="max-w-2xl w-full bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//             <CardHeader>
//               <div className="flex items-center gap-3 mb-4">
//                 <div className="p-3 bg-teal-500/10 rounded-lg">
//                   <AlertCircle className="w-8 h-8 text-teal-400" />
//                 </div>
//                 <div>
//                   <CardTitle className="text-2xl text-white">Welcome to Clutter!</CardTitle>
//                   <CardDescription className="text-gray-400">
//                     Let's set up your organization
//                   </CardDescription>
//                 </div>
//               </div>
//             </CardHeader>
//             <CardContent className="space-y-6">
//               <Alert className="bg-slate-800/50 border-slate-700">
//                 <AlertCircle className="h-4 w-4 text-teal-400" />
//                 <AlertTitle className="text-white">No Organization Found</AlertTitle>
//                 <AlertDescription className="text-gray-400">
//                   You need to create or join an organization before you can start creating projects.
//                 </AlertDescription>
//               </Alert>

//               <div className="space-y-4">
//                 <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
//                   <h3 className="text-white font-semibold mb-2">Create a New Organization</h3>
//                   <p className="text-sm text-gray-400 mb-4">
//                     Start fresh with your own organization. You'll be the owner and can invite team members later.
//                   </p>
//                   <Button 
//                     onClick={() => router.push('/onboarding/create-tenant')}
//                     className="w-full bg-teal-500 hover:bg-teal-600"
//                   >
//                     Create Organization
//                   </Button>
//                 </div>

//                 <div className="p-4 bg-slate-800/30 rounded-lg border border-slate-700">
//                   <h3 className="text-white font-semibold mb-2">Join an Existing Organization</h3>
//                   <p className="text-sm text-gray-400 mb-4">
//                     Have an invitation code? Join your team's organization here.
//                   </p>
//                   <Button 
//                     onClick={() => router.push('/onboarding/join-tenant')}
//                     variant="outline"
//                     className="w-full border-slate-600 text-white hover:bg-slate-800"
//                   >
//                     Join Organization
//                   </Button>
//                 </div>
//               </div>
//             </CardContent>
//           </Card>
//         </main>
//       </div>
//     );
//   }

//   // Has tenant - show dashboard
//   return (
//     <div className="min-h-screen bg-gradient-to-br from-black via-slate-900 to-teal-900 text-white">
//       <Navbar />
      
//       <main className="max-w-7xl mx-auto px-6 pt-32 pb-12">
//         {/* Header */}
//         <div className="flex items-center justify-between mb-8">
//           <div>
//             <h1 className="text-4xl font-bold mb-2">Overview</h1>
//             <p className="text-gray-400">
//               {userData?.tenant?.name || 'Your Organization'}
//             </p>
//           </div>
//           <Button 
//             onClick={() => router.push('/projects/new')}
//             className="bg-teal-500 hover:bg-teal-600 flex items-center gap-2"
//           >
//             <Plus className="w-4 h-4" />
//             New Project
//           </Button>
//         </div>

//         {error && (
//           <Alert className="mb-6 bg-red-900/20 border-red-800">
//             <AlertCircle className="h-4 w-4" />
//             <AlertTitle>Error</AlertTitle>
//             <AlertDescription>{error}</AlertDescription>
//           </Alert>
//         )}

//         {/* Stats Cards */}
//         <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
//           <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//             <CardHeader className="flex flex-row items-center justify-between pb-2">
//               <CardTitle className="text-sm font-medium text-gray-400">
//                 Total Projects
//               </CardTitle>
//               <FolderOpen className="w-4 h-4 text-teal-400" />
//             </CardHeader>
//             <CardContent>
//               <div className="text-3xl font-bold text-white">{projects.length}</div>
//             </CardContent>
//           </Card>

//           <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//             <CardHeader className="flex flex-row items-center justify-between pb-2">
//               <CardTitle className="text-sm font-medium text-gray-400">
//                 Recent Runs
//               </CardTitle>
//               <Activity className="w-4 h-4 text-teal-400" />
//             </CardHeader>
//             <CardContent>
//               <div className="text-3xl font-bold text-white">{recentRuns.length}</div>
//             </CardContent>
//           </Card>

//           <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//             <CardHeader className="flex flex-row items-center justify-between pb-2">
//               <CardTitle className="text-sm font-medium text-gray-400">
//                 Team Members
//               </CardTitle>
//               <Users className="w-4 h-4 text-teal-400" />
//             </CardHeader>
//             <CardContent>
//               <div className="text-3xl font-bold text-white">
//                 {projects.reduce((acc, p) => acc + (p.memberCount || 0), 0)}
//               </div>
//             </CardContent>
//           </Card>
//         </div>

//         {/* Projects Section */}
//         <div className="mb-8">
//           <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
//             <FolderOpen className="w-6 h-6 text-teal-400" />
//             Your Projects
//           </h2>
          
//           {projects.length === 0 ? (
//             <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//               <CardContent className="flex flex-col items-center justify-center py-12">
//                 <FolderOpen className="w-16 h-16 text-gray-600 mb-4" />
//                 <h3 className="text-xl font-semibold text-white mb-2">No projects yet</h3>
//                 <p className="text-gray-400 text-center mb-6">
//                   Create your first project to start building infrastructure
//                 </p>
//                 <Button 
//                   onClick={() => router.push('/projects/new')}
//                   className="bg-teal-500 hover:bg-teal-600"
//                 >
//                   <Plus className="w-4 h-4 mr-2" />
//                   Create Project
//                 </Button>
//               </CardContent>
//             </Card>
//           ) : (
//             <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
//               {projects.map((project) => (
//                 <Link key={project.projectId} href={`/projects/${project.projectId}`}>
//                   <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50 hover:border-teal-500/50 transition-all cursor-pointer group">
//                     <CardHeader>
//                       <CardTitle className="text-white group-hover:text-teal-400 transition-colors">
//                         {project.name}
//                       </CardTitle>
//                       <CardDescription className="text-gray-400 line-clamp-2">
//                         {project.description || 'No description'}
//                       </CardDescription>
//                     </CardHeader>
//                     <CardContent>
//                       <div className="flex items-center justify-between text-sm text-gray-500">
//                         <span className="flex items-center gap-1">
//                           <Clock className="w-4 h-4" />
//                           {formatDate(project.updatedAt)}
//                         </span>
//                         {project.memberCount && (
//                           <span className="flex items-center gap-1">
//                             <Users className="w-4 h-4" />
//                             {project.memberCount}
//                           </span>
//                         )}
//                       </div>
//                     </CardContent>
//                   </Card>
//                 </Link>
//               ))}
//             </div>
//           )}
//         </div>

//         {/* Recent Activity */}
//         <div>
//           <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
//             <Activity className="w-6 h-6 text-teal-400" />
//             Recent Activity
//           </h2>
          
//           {recentRuns.length === 0 ? (
//             <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//               <CardContent className="flex flex-col items-center justify-center py-12">
//                 <Activity className="w-16 h-16 text-gray-600 mb-4" />
//                 <h3 className="text-xl font-semibold text-white mb-2">No activity yet</h3>
//                 <p className="text-gray-400 text-center">
//                   Deploy your first workspace to see activity here
//                 </p>
//               </CardContent>
//             </Card>
//           ) : (
//             <Card className="bg-slate-900/50 backdrop-blur-xl border-slate-800/50">
//               <CardContent className="p-0">
//                 <div className="divide-y divide-slate-800">
//                   {recentRuns.map((run) => (
//                     <Link 
//                       key={run.runId} 
//                       href={`/projects/${run.projectId}/runs/${run.runId}`}
//                       className="block p-4 hover:bg-slate-800/30 transition-colors"
//                     >
//                       <div className="flex items-center justify-between">
//                         <div className="flex items-center gap-3">
//                           <div className={`w-2 h-2 rounded-full ${
//                             run.status === 'SUCCESS' ? 'bg-green-500' :
//                             run.status === 'FAILED' ? 'bg-red-500' :
//                             run.status === 'RUNNING' ? 'bg-blue-500 animate-pulse' :
//                             'bg-yellow-500'
//                           }`} />
//                           <div>
//                             <p className="text-white font-medium">{run.projectName}</p>
//                             <p className="text-sm text-gray-400">
//                               {run.action.toUpperCase()} • {run.workspaceId}
//                             </p>
//                           </div>
//                         </div>
//                         <div className="text-right">
//                           <p className={`text-sm font-medium ${getStatusColor(run.status)}`}>
//                             {run.status}
//                           </p>
//                           <p className="text-xs text-gray-500">
//                             {formatDate(run.startedAt)}
//                           </p>
//                         </div>
//                       </div>
//                     </Link>
//                   ))}
//                 </div>
//               </CardContent>
//             </Card>
//           )}
//         </div>
//       </main>

//       {/* Background Decoration */}
//       <div className="fixed inset-0 pointer-events-none opacity-20 -z-10">
//         <div className="absolute top-1/3 right-1/4 w-96 h-96 bg-teal-500 rounded-full filter blur-3xl"></div>
//         <div className="absolute bottom-1/3 left-1/4 w-96 h-96 bg-blue-500 rounded-full filter blur-3xl"></div>
//       </div>
//     </div>
//   );
// }

'use client';

import { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import DashboardLoading from './_components/DashboardLoading';
import DashboardOnboarding from './_components/DashboardOnboarding';
import DashboardContent from './_components/DashboardContent';

interface Project {
  projectId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  memberCount?: number;
}

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

interface UserData {
  userId: string;
  tenantId: string | null;
  email: string;
  displayName: string;
  tenant?: {
    tenantId: string;
    name: string;
  };
}

export default function DashboardPage() {
  const [userData, setUserData] = useState<UserData | null>(null);
  const [projects, setProjects] = useState<Project[]>([]);
  const [recentRuns, setRecentRuns] = useState<Run[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchDashboardData();
  }, []);

  const fetchDashboardData = async () => {
    try {
      setLoading(true);
      setError(null);

      const user = await apiClient.getUserProfile();
      setUserData(user);

      if (user.tenantId) {
        const [projectsData, runsData] = await Promise.all([
          apiClient.getProjects(),
          apiClient.getRecentRuns()
        ]);

        setProjects(projectsData.projects || []);
        setRecentRuns(runsData.runs || []);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  // State 1: Loading
  if (loading) {
    return <DashboardLoading />;
  }

  // State 2: No Tenant (Onboarding)
  if (userData && !userData.tenantId) {
    return <DashboardOnboarding />;
  }

  // State 3: Dashboard Content
  return (
    <DashboardContent
      userData={userData}
      projects={projects}
      recentRuns={recentRuns}
      error={error}
    />
  );
}