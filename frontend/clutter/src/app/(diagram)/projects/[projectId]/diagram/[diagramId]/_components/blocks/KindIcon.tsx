import { Archive, Box, Globe, HardDrive, Network, Server, Shield, Table2, Zap } from "lucide-react";

// The icon for a building block. A kind a cloud adds on its own gets a plain box.
export default function KindIcon({ kind, className }: { kind: string; className?: string }) {
  switch (kind) {
    case "vm":
      return <Server className={className} />;
    case "network":
      return <Network className={className} />;
    case "firewall":
      return <Shield className={className} />;
    case "volume":
      return <HardDrive className={className} />;
    case "bucket":
      return <Archive className={className} />;
    case "function":
      return <Zap className={className} />;
    case "table":
      return <Table2 className={className} />;
    case "http_api":
      return <Globe className={className} />;
    default:
      return <Box className={className} />;
  }
}
