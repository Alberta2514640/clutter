"use client";
import PresenceBar from "@/components/common/PresenceBar";
import Canvas from "@/components/diagram/Canvas";
import Console from "@/components/diagram/Console";

export default function DiagramPage() {
  return (
    <div className="flex flex-col">
      <PresenceBar />
      <div className="p-4">
        <Canvas />
        <div className="mt-4">
          <Console />
        </div>
      </div>
    </div>
  );
}
