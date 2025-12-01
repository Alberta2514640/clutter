import NextAuth from "next-auth";
import GoogleProvider from "next-auth/providers/google";

const handler = NextAuth({
  providers: [
    GoogleProvider({
      clientId: process.env.GOOGLE_CLIENT_ID!,
      clientSecret: process.env.GOOGLE_CLIENT_SECRET!,
    }),
  ],
  // optional, but good to have:
  // callbacks: {
  //   async session({ session, token }) {
  //     // attach stuff to session if needed
  //     return session;
  //   },
  // },
});


export { handler as GET, handler as POST };
