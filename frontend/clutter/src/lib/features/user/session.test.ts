import { describe, expect, it } from "vitest";
import { ApiError, apiErrorMessage } from "./session";

describe("apiErrorMessage", () => {
  it("prefers the specific reason, then the message", () => {
    const body = JSON.stringify({ message: "something went wrong", error: "accountName must be between 4 and 32 characters" });
    expect(apiErrorMessage(new ApiError(400, body))).toBe("accountName must be between 4 and 32 characters");
    expect(apiErrorMessage(new ApiError(400, JSON.stringify({ message: "invalid AWS region" })))).toBe("invalid AWS region");
  });

  it("gives null for a body that is not A's JSON, or an error that is not A's", () => {
    expect(apiErrorMessage(new ApiError(502, "Bad Gateway"))).toBeNull();
    expect(apiErrorMessage(new ApiError(400, JSON.stringify({ message: "" })))).toBeNull();
    expect(apiErrorMessage(new Error(JSON.stringify({ message: "x" })))).toBeNull();
    expect(apiErrorMessage(null)).toBeNull();
  });
});
