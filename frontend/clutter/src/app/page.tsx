import Navbar from "@/components/common/Navbar";
import Link from "next/link";
import { HeroGeometric } from "@/components/ui/shape-landing-hero";
import { Button } from "@/components/ui/button";
import { Cloud, Boxes, GitBranch, Workflow, ShieldCheck, Users2, Cpu, ArrowRight } from "lucide-react";

export default function Home() {
  return (
    <div className="min-h-screen bg-black text-white flex flex-col">
      <Navbar />

      <main className="flex-1">
        {/* Hero */}
        <HeroGeometric badge="Clutter · Infrastructure Diagramming Studio" title1="Infrastructure" title2="simplified" />

        {/* Section: What is Clutter */}
        <section className="border-t border-slate-800/60 bg-gradient-to-b from-black via-slate-950 to-black">
          <div className="container mx-auto px-4 md:px-8 py-16 md:py-24">
            <div className="max-w-3xl mb-10">
              <p className="text-sm font-semibold tracking-[0.2em] text-cyan-400 uppercase mb-3">Overview</p>
              <h2 className="text-3xl md:text-4xl font-semibold tracking-tight mb-4">From cloud diagram to production-ready Infrastructure-as-Code.</h2>
              <p className="text-slate-300/80 leading-relaxed">
                Clutter is a diagram-first Infrastructure-as-Code editor. Draw AWS architectures, connect services with validated rules, and let Clutter generate Terraform and Ansible for you. Under the hood, everything is modeled around tenants,
                projects, canvases, nodes, edges, workspaces, and runs — so your diagrams are never just pictures, they&apos;re deployable artifacts.
              </p>
            </div>

            {/* Feature grid */}
            <div className="grid gap-6 md:grid-cols-3">
              <div className="rounded-2xl border border-slate-800/80 bg-slate-950/60 p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 rounded-xl bg-teal-500/10 border border-teal-500/40">
                    <Cloud className="h-5 w-5 text-teal-400" />
                  </div>
                  <h3 className="font-semibold text-lg">Diagram → Terraform</h3>
                </div>
                <p className="text-sm text-slate-300/80 mb-3">Drag AWS resources onto a canvas and connect them with typed rules. Clutter turns the resulting graph into Terraform modules, aligned with your approved ModuleSets and VarSets.</p>
                <ul className="space-y-1 text-xs text-slate-400">
                  <li>• React Flow–powered canvas with snap grid</li>
                  <li>• Deterministic connection rules &amp; cycle prevention</li>
                  <li>• Export Terraform bundles via a single click</li>
                </ul>
              </div>

              <div className="rounded-2xl border border-slate-800/80 bg-slate-950/60 p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 rounded-xl bg-indigo-500/10 border border-indigo-500/40">
                    <Boxes className="h-5 w-5 text-indigo-300" />
                  </div>
                  <h3 className="font-semibold text-lg">Workspaces &amp; Runs</h3>
                </div>
                <p className="text-sm text-slate-300/80 mb-3">Bind diagrams to execution contexts. Workspaces point to AWS accounts and module catalogs, while Runs orchestrate plan/apply lifecycles with live logs and status.</p>
                <ul className="space-y-1 text-xs text-slate-400">
                  <li>• Workspace-aware editing and environment VarSets</li>
                  <li>• Plan / apply flows with run status and log streaming</li>
                  <li>• Run history linked back to each project</li>
                </ul>
              </div>

              <div className="rounded-2xl border border-slate-800/80 bg-slate-950/60 p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 rounded-xl bg-rose-500/10 border border-rose-500/40">
                    <GitBranch className="h-5 w-5 text-rose-300" />
                  </div>
                  <h3 className="font-semibold text-lg">Single-table core</h3>
                </div>
                <p className="text-sm text-slate-300/80 mb-3">A DynamoDB single-table design powers everything: tenants, projects, canvases, nodes, edges, workspaces, module sets, var sets, and runs — optimized for real-world access patterns.</p>
                <ul className="space-y-1 text-xs text-slate-400">
                  <li>• Tenant-scoped multi-tenancy</li>
                  <li>• Membership roles: owner, editor, viewer</li>
                  <li>• GSIs tuned for common queries &amp; dashboards</li>
                </ul>
              </div>
            </div>
          </div>
        </section>

        {/* Section: How it works */}
        <section className="border-t border-slate-800/60 bg-black">
          <div className="container mx-auto px-4 md:px-8 py-16 md:py-24">
            <div className="grid gap-10 md:grid-cols-[1.2fr,1fr] items-start">
              <div>
                <p className="text-sm font-semibold tracking-[0.2em] text-cyan-400 uppercase mb-3">Flow</p>
                <h2 className="text-3xl md:text-4xl font-semibold tracking-tight mb-4">A focused, opinionated workflow.</h2>
                <p className="text-slate-300/80 leading-relaxed mb-8">
                  Clutter&apos;s front-end mirrors the backend&apos;s entities and access patterns. From loading a diagram to streaming deployment logs, each step is designed to feel fast, predictable, and safe.
                </p>

                <ol className="space-y-5 text-sm text-slate-200">
                  <li className="flex gap-3">
                    <span className="mt-0.5 h-6 w-6 rounded-full bg-teal-500/20 border border-teal-500/60 flex items-center justify-center text-xs">1</span>
                    <div>
                      <h3 className="font-medium mb-1">Load a project &amp; canvas</h3>
                      <p className="text-slate-400">Select a project, then open a diagram. The client fetches a CanvasBundle — canvas + nodes + edges — in one request, backed by GSI2. React Flow hydrates the graph in milliseconds.</p>
                    </div>
                  </li>

                  <li className="flex gap-3">
                    <span className="mt-0.5 h-6 w-6 rounded-full bg-indigo-500/20 border border-indigo-500/60 flex items-center justify-center text-xs">2</span>
                    <div>
                      <h3 className="font-medium mb-1">Edit with autosave &amp; validation</h3>
                      <p className="text-slate-400">
                        Drag nodes, create edges, and tweak specs. Changes go through a local history stack (undo/redo) and optimistic PATCH calls with ETag-based conflict handling. Connection rules and zod schemas prevent broken graphs.
                      </p>
                    </div>
                  </li>

                  <li className="flex gap-3">
                    <span className="mt-0.5 h-6 w-6 rounded-full bg-rose-500/20 border border-rose-500/60 flex items-center justify-center text-xs">3</span>
                    <div>
                      <h3 className="font-medium mb-1">Plan &amp; apply safely</h3>
                      <p className="text-slate-400">
                        Bind the diagram to a Workspace, then trigger a Run. The backend composes Terraform from the graph, assumes an AWS role through AccountLinks, and streams plan/apply logs back to the UI with clear status chips and summaries.
                      </p>
                    </div>
                  </li>
                </ol>
              </div>

              {/* Side card: Tech stack */}
              <div className="rounded-2xl border border-slate-800/80 bg-slate-950/80 p-6 md:p-7">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-2 rounded-xl bg-slate-800/80 border border-slate-700">
                    <Cpu className="h-5 w-5 text-slate-100" />
                  </div>
                  <h3 className="font-semibold text-lg">Front-end stack</h3>
                </div>
                <ul className="space-y-2 text-sm text-slate-300/90">
                  <li>• Next.js + React + TypeScript</li>
                  <li>• React Flow for diagramming</li>
                  <li>• Zustand for canvas state + history</li>
                  <li>• React Query for data fetching &amp; caching</li>
                  <li>• Yjs + y-websocket for collaboration &amp; presence</li>
                  <li>• Tailwind + shadcn/ui + Lucide Icons</li>
                </ul>
                <p className="text-xs text-slate-500 mt-4">Every UI concept — from Canvas to Workspace to Run — maps directly to a backend entity, keeping the client thin and predictable.</p>
              </div>
            </div>
          </div>
        </section>

        {/* Section: Collaboration & safety */}
        <section className="border-t border-slate-800/60 bg-gradient-to-b from-black via-slate-950 to-black">
          <div className="container mx-auto px-4 md:px-8 py-16 md:py-20">
            <div className="flex flex-col md:flex-row gap-10 items-start md:items-center">
              <div className="flex-1">
                <p className="text-sm font-semibold tracking-[0.2em] text-cyan-400 uppercase mb-3">Teams</p>
                <h2 className="text-3xl md:text-4xl font-semibold tracking-tight mb-4">Built for real teams, not just solo diagrams.</h2>
                <p className="text-slate-300/80 leading-relaxed mb-6">Clutter ships with opinionated collaboration and safety features so your diagrams stay consistent and your deployments stay boring (in a good way).</p>

                <div className="grid gap-4 md:grid-cols-2">
                  <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <Users2 className="h-4 w-4 text-teal-300" />
                      <span className="text-sm font-medium">Presence &amp; locking</span>
                    </div>
                    <p className="text-xs text-slate-400">See who&apos;s viewing a canvas in real time. A Git-style single-editor lock ensures only one editor can modify a diagram while others stay safely in read-only mode.</p>
                  </div>

                  <div className="rounded-xl border border-slate-800 bg-slate-950/60 p-4">
                    <div className="flex items-center gap-2 mb-2">
                      <ShieldCheck className="h-4 w-4 text-emerald-300" />
                      <span className="text-sm font-medium">Guard rails &amp; validation</span>
                    </div>
                    <p className="text-xs text-slate-400">Connection rules, schema validation, and conflict dialogs keep your architecture clean. Secrets are never stored client-side and logs are redacted by design.</p>
                  </div>
                </div>
              </div>

              {/* CTA block */}
              <div className="w-full md:w-auto rounded-2xl border border-cyan-500/40 bg-cyan-500/5 px-6 py-6 md:px-8 md:py-8 shadow-[0_0_40px_rgba(45,212,191,0.15)]">
                <div className="flex items-center gap-2 mb-3">
                  <Workflow className="h-5 w-5 text-cyan-300" />
                  <span className="text-sm font-medium text-cyan-200">Ready to start diagramming?</span>
                </div>
                <p className="text-sm text-slate-200 mb-5">Sign in, create a project, and draw your first AWS architecture. Clutter will handle the Terraform and deployments for you.</p>
                <div className="flex flex-col sm:flex-row gap-3">
                  <Button asChild className="bg-cyan-500 hover:bg-cyan-400 text-black font-medium px-6">
                    <Link href="/login" className="flex items-center gap-2">
                      Get started
                      <ArrowRight className="h-4 w-4" />
                    </Link>
                  </Button>
                  <Button asChild variant="outline" className="border-white/20 text-black/80 hover:bg-white/10 hover:border-black/40 hover:text-white/80 px-6">
                    <Link href="/docs">View docs</Link>
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-black/80">
        <div className="container mx-auto px-4 md:px-8 py-6 flex flex-col md:flex-row items-center justify-between gap-3 text-xs text-slate-500">
          <span>© {new Date().getFullYear()} Clutter. All rights reserved.</span>
          <div className="flex gap-4">
            <Link href="/terms" className="hover:text-slate-300">
              Terms
            </Link>
            <Link href="/privacy" className="hover:text-slate-300">
              Privacy
            </Link>
          </div>
        </div>
      </footer>
    </div>
  );
}
