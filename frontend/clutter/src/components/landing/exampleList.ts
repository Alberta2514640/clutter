import type { ExampleLevel } from "@/lib/features/examples/types";

// The built-in examples as the home page lists them. B serves them only to a
// signed-in caller (GET /v1/examples), and the home page is public, so this is
// a copy. Its owner is backend/adapter/internal/examples/*.json: exampleList.test.ts
// reads those files and fails when this list drifts from them. The page never
// states how many there are.
export type LandingExample = {
  id: string;
  name: string;
  level: ExampleLevel;
  description: string;
  // Connector ids, as in the example file.
  clouds: string[];
  // One deployment over all its clouds (the file has placements).
  split: boolean;
};

export const LANDING_EXAMPLES: LandingExample[] = [
  {
    id: "single-server",
    name: "Single server",
    level: "starter",
    description: "One server you can reach, in its own network.",
    clouds: ["aws", "general-openstack"],
    split: false,
  },
  {
    id: "file-storage",
    name: "File storage",
    level: "starter",
    description: "One bucket that keeps old versions of your files.",
    clouds: ["aws"],
    split: false,
  },
  {
    id: "web-server",
    name: "Web server",
    level: "intermediate",
    description: "A public VM that serves your site on port 80, behind a firewall.",
    clouds: ["aws", "general-openstack"],
    split: false,
  },
  {
    id: "serverless-api",
    name: "Serverless API",
    level: "intermediate",
    description: "An HTTP API that runs a function, which reads a table. It answers out of the box.",
    clouds: ["aws"],
    split: false,
  },
  {
    id: "two-tier-app",
    name: "Two-tier app",
    level: "advanced",
    description: "A web server and a private database server that can find each other.",
    clouds: ["aws", "general-openstack"],
    split: false,
  },
  {
    id: "full-app",
    name: "Full app",
    level: "advanced",
    description: "A two-tier app with file storage and an API.",
    clouds: ["aws"],
    split: false,
  },
  {
    id: "hybrid",
    name: "Hybrid",
    level: "advanced",
    description: "A VM on OpenStack that reads a bucket on AWS, in one deployment.",
    clouds: ["general-openstack", "aws"],
    split: true,
  },
];
