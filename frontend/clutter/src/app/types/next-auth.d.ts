import "next-auth";

declare module "next-auth" {
  interface Session {
    accessToken?: string;
    idToken?: string;
    provider?: string;
  }

  interface User {
    // you can add more stuff here later (e.g. tenantId, roles)
    id: string;
    email: string;
    name: string;
    tenantId: string;
    role: "owner" | "editor" | "viewer";
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    idToken?: string;
    provider?: string;
  }
}
