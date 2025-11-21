"use client";

import { useState } from "react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  Avatar,
  AvatarFallback,
} from "@/components/ui/avatar";

import { Layers } from "lucide-react";

export default function ProjectSettingsPage() {
  const [projectName, setProjectName] = useState("My project");
  const [description, setDescription] = useState("");

  return (
    <div className="max-w-4xl mx-auto py-12 space-y-12 text-white">

      {/* ICON + NAME FIELDSET */}
      <div className="space-y-3">
        <Label className="text-gray-300">Icon and name</Label>

        <div className="flex items-center gap-4">
          {/* Icon button (matches your existing UI) */}
          <Button
            variant="outline"
            className="w-12 h-12 rounded-xl bg-slate-800 border-slate-700 text-gray-300 hover:border-teal-400"
          >
            <Layers className="w-5 h-5" />
          </Button>

          {/* Project name input */}
          <Input
            className="bg-slate-800 border-slate-700 text-white"
            value={projectName}
            onChange={(e) => setProjectName(e.target.value)}
          />
        </div>
      </div>

      {/* DESCRIPTION FIELDSET */}
      <div className="space-y-3">
        <Label className="text-gray-300">Description</Label>

        <Textarea
          className="bg-slate-800 border-slate-700 text-white"
          rows={3}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>

      


      {/* DANGER ZONE */}
      <div className="space-y-4">
        <h3 className="text-lg font-semibold text-red-400">Danger zone</h3>
        <p className="text-gray-400 text-sm">
          When deleting a project, you can also choose to move all workflows and
          credentials to another project.
        </p>

        <Button className="bg-red-600 hover:bg-red-700 text-white">
          Delete this project
        </Button>
      </div>
    </div>
  );
}
