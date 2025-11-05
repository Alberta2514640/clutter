"use client";

export default function PresenceBar() {
  return (
    <div className="h-8 border-b px-3 text-sm flex items-center justify-between bg-muted/40">
      <span className="font-medium text-muted-foreground">Presence</span>
      {/* Later: add collaborator avatars or cursors here */}
    </div>
  );
}
