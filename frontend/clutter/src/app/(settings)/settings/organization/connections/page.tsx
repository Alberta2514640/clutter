import type { Metadata } from "next";
import ConnectionsPage from "./_components/ConnectionsPage";

export const metadata: Metadata = { title: "Connections" };

// ?connect=<connector> opens the connection flow on that cloud.
export default async function OrganizationConnectionsPage({ searchParams }: { searchParams: Promise<{ [key: string]: string | string[] | undefined }> }) {
  const connect = (await searchParams).connect;
  return <ConnectionsPage openConnector={typeof connect === "string" && connect ? connect : null} />;
}
