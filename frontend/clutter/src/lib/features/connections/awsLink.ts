// The choices the AWS connection flow makes, kept apart from the form so they
// can be tested. AWS is linked through A's CloudFormation flow: Clutter opens
// a stack in the user's account (an unfinished link), the user pastes the
// role ARN the stack outputs (the link is complete), then B connects the
// linked account. B's linked-account id is A's link id
// (backend/adapter/connectors/aws/linked.go), so a complete link is
// connected by its own id.

// Relative imports: Vitest runs this file without the "@/" alias.
import { AdapterApiError, adapterErrorMessage } from "../../adapterApi";
import type { OrganizationAwsAccount } from "../organization/types";
import { awsRoleArnExample, awsRoleArnPattern } from "../organization/rules";

// roleArnProblem says what is wrong with a pasted role ARN, or null. An empty
// field has no problem yet; the button stays off until the ARN is valid.
export function roleArnProblem(value: string): string | null {
  const v = value.trim();
  if (v === "") return null;
  if (awsRoleArnPattern.test(v)) return null;
  if (!v.startsWith("arn:aws:iam::")) return `A role ARN starts with arn:aws:iam::. Copy RoleArn from the stack's Outputs tab.`;
  return `This is not the role Clutter's stack creates. It looks like ${awsRoleArnExample}.`;
}

// accountsInOrder lists the organization's links in a stable order (A's list
// has none): linked first, then unfinished, each by name.
export function accountsInOrder(accounts: OrganizationAwsAccount[]): OrganizationAwsAccount[] {
  const rank = (a: OrganizationAwsAccount) => (a.status === "complete" ? 0 : a.status === "incomplete" ? 1 : 2);
  return [...accounts].sort((a, b) => rank(a) - rank(b) || a.account_name.localeCompare(b.account_name) || a.id.localeCompare(b.id));
}

// resumeTarget is the unfinished link the flow continues: the one the user
// chose, else the first unfinished one.
export function resumeTarget(accounts: OrganizationAwsAccount[], chosenId: string | null): OrganizationAwsAccount | null {
  const unfinished = accountsInOrder(accounts).filter((a) => a.status === "incomplete");
  return unfinished.find((a) => a.id === chosenId) ?? unfinished[0] ?? null;
}

// The devserver's Link fake account route links a role in AWS's documentation
// account (FakeRoleArn in backend/devserver/internal/gateway/fakeaccount.go).
// No real link uses that account.
const FAKE_ROLE_PREFIX = "arn:aws:iam::123456789012:role/";

export function isFakeLocalRole(roleArn: string | null | undefined): boolean {
  return !!roleArn && roleArn.startsWith(FAKE_ROLE_PREFIX);
}

// What the flow says when B could not use a fake local account: Clutter on
// this machine is set to build in real AWS, where that role does not exist.
// B's own message would tell the user their CloudFormation stack was deleted.
export const LOCAL_AWS_REFUSED =
  "This is a fake local AWS account, and Clutter on this machine is set up to build in real AWS, where it does not exist. Link a real AWS account below, or switch this machine back to LocalStack (backend/devserver/README.md).";

// connectFailure is one line for a refused connection. fakeLocal says the
// account is a fake local one, whose refusal has its own words.
export function connectFailure(err: unknown, fakeLocal: boolean): string {
  if (fakeLocal && err instanceof AdapterApiError && err.code === "verification_failed") return LOCAL_AWS_REFUSED;
  return adapterErrorMessage(err);
}
