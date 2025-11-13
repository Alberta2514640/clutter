import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone", //need for docker deployment
  reactCompiler: true,
};

export default nextConfig;
