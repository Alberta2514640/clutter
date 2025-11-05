"use client";

import { Button } from "@/components/ui/button";
import { Github, Cloud, Boxes, ArrowRight } from "lucide-react";
import Link from "next/link";

export default function Home() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-background to-muted px-6 text-center">
      <div className="max-w-3xl space-y-6">
        <div className="flex flex-col items-center gap-2">
          <Boxes className="h-10 w-10 text-primary" />
          <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">Clutter</h1>
          <p className="text-lg text-muted-foreground max-w-xl">Visualize your cloud architecture. Build and deploy Terraform + Ansible configurations directly from your diagram.</p>
        </div>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-3 pt-2">
          <Link href="/projects">
            <Button size="lg" className="group">
              Get Started
              <ArrowRight className="ml-2 h-4 w-4 transition-transform group-hover:translate-x-1" />
            </Button>
          </Link>

          <Button variant="outline" size="lg" asChild>
            <a href="https://github.com/" target="_blank" rel="noopener noreferrer" className="flex items-center gap-2">
              <Github className="h-4 w-4" />
              GitHub
            </a>
          </Button>
        </div>

        <div className="pt-8 text-sm text-muted-foreground">
          <Cloud className="inline h-4 w-4 mr-1 text-primary/70" />
          Open source • Built with Next.js, React Flow, and AWS
        </div>
      </div>
    </main>
  );
}
