import { redirect } from "next/navigation";

// AWS accounts are linked on Connections, in the one connection flow. This
// address stays for old links and A's editor, and opens that flow on AWS.
export default function OrganizationAwsAccountPage() {
  redirect("/settings/organization/connections?connect=aws");
}
