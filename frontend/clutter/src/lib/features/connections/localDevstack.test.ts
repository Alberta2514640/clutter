import { describe, expect, it } from "vitest";
import { localDevApi } from "../organization/localDev";
import { APP_CRED_ID, APP_CRED_SECRET, AUTH_URL } from "./cloudsYaml";
import { offersLocalDevstack } from "./localDevstack";

const openstack = [AUTH_URL, APP_CRED_ID, APP_CRED_SECRET];

describe("offersLocalDevstack", () => {
  it("offers the button on an OpenStack credential form in a local dev build", () => {
    expect(offersLocalDevstack(localDevApi("development", "http://localhost:8787"), openstack, "credentials", false)).toBe(true);
  });

  it("never offers it in a production build, whatever the API address", () => {
    for (const api of ["http://localhost:8787", "https://api.clutter.example"]) {
      expect(offersLocalDevstack(localDevApi("production", api), openstack, "credentials", false)).toBe(false);
    }
  });

  it("never offers it against an API that is not the local gateway", () => {
    expect(offersLocalDevstack(localDevApi("development", "https://api.clutter.example"), openstack, "credentials", false)).toBe(false);
  });

  it("offers it only where DevStack's credential fits the form", () => {
    expect(offersLocalDevstack(true, [AUTH_URL, APP_CRED_ID], "credentials", false)).toBe(false);
    expect(offersLocalDevstack(true, ["access_key", "secret_key"], "credentials", false)).toBe(false);
    expect(offersLocalDevstack(true, openstack, "linked", false)).toBe(false);
    expect(offersLocalDevstack(true, openstack, "credentials", true)).toBe(false);
  });
});
