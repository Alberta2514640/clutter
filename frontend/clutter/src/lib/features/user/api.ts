import type { GoogleDataShape, UserData } from "./types";

//Calling API env var from env
const API_BASE = process.env.NEXT_PUBLIC_API_ENDPOINT;

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

/* ---------------------------------------------
 * Local storage helpers (snake_case persistence)
 * -------------------------------------------- */

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

/* ---------------------------------------------
 * Mapping: snake_case → camelCase
 * -------------------------------------------- */

const toUserData = (g: GoogleDataShape): UserData => {
  const u = g.user_data ?? {};

  return {
    userId: u.uuid ?? "u_unknown",
    displayName: u.full_name ?? "Unknown User",
    email: u.email ?? "unknown@example.com",
    pictureUrl: u.picture_url ?? "",
    token: g.token ?? "",
  };
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
  loginWithGoogle: async (google_id_token: string): Promise<UserData> => {
    if (!google_id_token) {
      throw new Error("Missing Google ID token");
    }

    const res = await fetch(`${API_BASE}/log-in`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token: google_id_token }),
    });

    if (!res.ok) {
      const text = await res.text().catch(() => "");
      throw new Error(`Login failed (${res.status}): ${text || res.statusText}`);
    }

    const data = (await res.json()) as GoogleDataShape;

    writeGoogleData(data);
    return toUserData(data);
  },

  getMe: async (): Promise<UserData | null> => {
    await sleep(250);

    const gData = readGoogleData();
    if (!gData) return null; //null since logout, no user, is not at error

    return toUserData(gData);
  },

  setMe: async (user: Partial<UserData>): Promise<UserData> => {
    await sleep(250);

    const gData: GoogleDataShape = readGoogleData() ?? {};
    const uData = gData.user_data ?? {};

    if (user.userId) uData.uuid = user.userId;
    if (user.email) uData.email = user.email;
    if (user.displayName) uData.full_name = user.displayName;
    if (user.pictureUrl) uData.picture_url = user.pictureUrl;
    if (user.token) gData.token = user.token;

    gData.user_data = uData;
    writeGoogleData(gData);

    return toUserData(gData);
  },
};
