import { describe, expect, it } from "vitest";
import { errorFromBody } from "./adapterApi";

describe("errorFromBody", () => {
  it("reads B's envelope with its code and problems", () => {
    const body = JSON.stringify({
      error: { code: "invalid_code", message: "This file is not a zip.", problems: [{ node_id: "fn", code: "bad_zip", message: "Not a zip." }] },
    });
    const err = errorFromBody(422, body, "The upload failed (422).");
    expect(err.status).toBe(422);
    expect(err.code).toBe("invalid_code");
    expect(err.message).toBe("This file is not a zip.");
    expect(err.problems).toEqual([{ node_id: "fn", code: "bad_zip", message: "Not a zip." }]);
  });

  it("gives the fallback for a body that is not the envelope", () => {
    for (const body of ["", "<html>Bad gateway</html>", JSON.stringify({ message: "x" })]) {
      const err = errorFromBody(502, body, "The request failed (502).");
      expect([err.code, err.message, err.problems]).toEqual(["error", "The request failed (502).", []]);
    }
  });
});
