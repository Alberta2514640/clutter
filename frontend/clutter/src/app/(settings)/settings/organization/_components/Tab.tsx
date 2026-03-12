import Link from "next/link";
import { usePathname } from "next/navigation";

export default function Tabs() {
  const pathname = usePathname();
  const isManage = pathname?.endsWith("/manage");
  const isAws = pathname?.endsWith("/aws");

  const tabBase = "px-4 py-2 text-m font-medium border-b-2 -mb-px transition-colors";
  const inactiveTab = tabBase + " border-transparent text-slate-400 hover:text-slate-200 hover:border-slate-600";
  const activeTab = tabBase + " border-teal-400 text-slate-50";

  return (
    <nav className="border-b border-slate-800 mb-6 flex gap-4 text-m">
      <Link href="/settings/organization/manage" className={isManage ? activeTab : inactiveTab}>
        Manage
      </Link>
      <Link href="/settings/organization/aws" className={isAws ? activeTab : inactiveTab}>
        AWS Account
      </Link>
    </nav>
  );
}