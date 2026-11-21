import ThemeScope from "@/components/common/ThemeScope";

// Sign-in and onboarding are light whatever the theme. A dark part inside one
// (the sign-in brand panel) sets its own `dark` scope.
export default function AuthGroupLayout({ children }: { children: React.ReactNode }) {
  return (
    <ThemeScope theme="light" className="bg-background text-foreground min-h-dvh">
      {children}
    </ThemeScope>
  );
}
