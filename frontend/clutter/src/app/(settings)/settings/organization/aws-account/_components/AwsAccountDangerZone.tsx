import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";

interface AwsAccountDangerZoneProps {
  accountName: string;
  isDeleting: boolean;
  onDelete: () => Promise<void>;
}

export default function AwsAccountDangerZone({ accountName, isDeleting, onDelete }: AwsAccountDangerZoneProps) {
  return (
    <Card className="border-red-900/60 bg-red-950/20 shadow-none">
      <CardHeader>
        <CardTitle className="text-xl text-red-300">Danger zone</CardTitle>
        <CardDescription className="text-red-200/70">
          Deleting this AWS account link removes it from the organization and returns this page to the account creation flow.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-slate-300">
          Delete <span className="font-semibold text-white">{accountName}</span> if you need to recreate the link from scratch.
        </p>
        <Button
          type="button"
          onClick={() => void onDelete()}
          disabled={isDeleting}
          className="bg-red-600 text-white hover:bg-red-500"
        >
          {isDeleting ? "Deleting account..." : "Delete account link"}
        </Button>
      </CardContent>
    </Card>
  );
}
