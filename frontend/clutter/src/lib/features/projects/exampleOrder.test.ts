import { describe, expect, it } from "vitest";
import type { Example } from "../examples/types";
import { orderExamplesForCloud } from "./exampleOrder";

const ex = (id: string, clouds: string[]): Example => ({ id, name: id, level: "starter", description: "", clouds, document: {} });

describe("orderExamplesForCloud", () => {
  const list = [ex("a", ["aws"]), ex("b", ["general-openstack"]), ex("c", ["aws", "general-openstack"]), ex("d", ["general-openstack"])];

  it("puts the examples that fit the cloud first, keeping B's order", () => {
    expect(orderExamplesForCloud(list, "general-openstack").map((e) => e.id)).toEqual(["b", "c", "d", "a"]);
    expect(orderExamplesForCloud(list, "aws").map((e) => e.id)).toEqual(["a", "c", "b", "d"]);
  });

  it("keeps B's order without a cloud", () => {
    expect(orderExamplesForCloud(list, null).map((e) => e.id)).toEqual(["a", "b", "c", "d"]);
  });
});
