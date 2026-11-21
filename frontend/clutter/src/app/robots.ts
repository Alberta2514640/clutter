import type { MetadataRoute } from "next";
import { SITE_URL } from "@/components/landing/site";

// Crawlers may read the public pages and what they load (images, scripts, the
// Open Graph image), and nothing of the signed-in app. "/$" is the home page
// alone; the longer Allow rules win over "Disallow: /".
export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: ["/$", "/privacy", "/terms", "/landing/", "/logos/", "/opengraph-image", "/_next/"],
      disallow: "/",
    },
    sitemap: `${SITE_URL}/sitemap.xml`,
  };
}
