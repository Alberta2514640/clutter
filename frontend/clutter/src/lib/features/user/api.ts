// src/lib/features/user/api.ts
import type { UserData } from "./types";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

type GoogleDataShape = {
  token?: string;
  user_data?: {
    uuid?: string;
    email?: string;
    full_name?: string;
    picture_url?: string;
  };
};

const readGoogleData = (): GoogleDataShape | null => {
  if (typeof window === "undefined") return null;

  const raw = window.localStorage.getItem("google_data");
  if (!raw) return null;

  try {
    return JSON.parse(raw) as GoogleDataShape;
  } catch {
    return null;
  }
};

const writeGoogleData = (obj: GoogleDataShape) => {
  if (typeof window === "undefined") return;
  window.localStorage.setItem("google_data", JSON.stringify(obj));
};

const toUserData = (g: GoogleDataShape): UserData => {
  const u = g.user_data ?? {};
  return {
    userId: u.uuid ?? "u_unknown",
    displayName: u.full_name ?? "Unknown User",
    email: u.email ?? "unknown@example.com",
    token: g.token ?? "",
    picture_url: u.picture_url ?? "",
  };
};

// optional: centralize your API base
const API_BASE = "https://qzq3ncab46.execute-api.us-west-2.amazonaws.com/prod";

export const userApi = {
  loginWithGoogle: async (googleIdToken: string): Promise<UserData> => {
    // no sleep here; this is a real network call
    if (!googleIdToken) throw new Error("Missing Google ID token");

    const res = await fetch(`${API_BASE}/log-in`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token: googleIdToken }),
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(`Login failed (${res.status}): ${text || res.statusText}`);
    }

    // backend returns the object you were storing
    const data = (await res.json()) as GoogleDataShape;

    // persist for later getMe()
    writeGoogleData(data);

    // return typed user info for immediate use
    return toUserData(data);
  },

  getMe: async (): Promise<UserData> => {
    await sleep(250);
    const gData = readGoogleData();
    if (!gData) throw new Error("No Google data found in local storage");
    return toUserData(gData);
  },

  setMe: async (userData: Partial<UserData>): Promise<UserData> => {
    await sleep(250);
    const gData = readGoogleData() || {};
    const uData = gData.user_data || {};

    if (userData.userId) uData.uuid = userData.userId;
    if (userData.email) uData.email = userData.email;
    if (userData.displayName) uData.full_name = userData.displayName;
    if (userData.picture_url) uData.picture_url = userData.picture_url;
    if (userData.token) gData.token = userData.token;

    gData.user_data = uData;
    writeGoogleData(gData);

    return toUserData(gData);
  },
};
