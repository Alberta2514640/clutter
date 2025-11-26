// import NextAuth from "next-auth";
// import CognitoProvider from "next-auth/providers/cognito";

// const handler = NextAuth({
//   providers: [
//     CognitoProvider({
//       clientId: process.env.COGNITO_CLIENT_ID!,
//       clientSecret: process.env.COGNITO_CLIENT_SECRET!,
//       issuer: `https://cognito-idp.${process.env.COGNITO_REGION}.amazonaws.com/${process.env.COGNITO_USER_POOL_ID}`,
//     }),
//   ],
// });

// export { handler as GET, handler as POST };

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
