import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", //need for docker deployment
  reactCompiler: true,
  // next dev would otherwise write AGENTS.md into this folder on every start.
  agentRules: false,
};

export default nextConfig;
