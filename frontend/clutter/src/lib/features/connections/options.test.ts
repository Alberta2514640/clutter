import { describe, expect, it } from "vitest";
import type { FieldSpec } from "../catalog/types";
import { applySuggestions, optionsState, settingView, type OptionsState } from "./options";
import type { ConnectionOptions } from "./types";

// The OpenStack connector's settings, as its manifest declares them.
const image: FieldSpec = { name: "image", title: "Image", type: "string", required: true, default: "cirros" };
const flavor: FieldSpec = { name: "flavor_small", title: "Flavor for small VMs", type: "string", required: true, default: "m1.tiny" };
const network: FieldSpec = { name: "external_network", title: "External network", type: "string", required: true, default: "public" };
const size: FieldSpec = { name: "size", title: "Size", type: "enum", required: true, allowed: ["small", "large"] };

const ready: OptionsState = {
  status: "ready",
  options: {
    settings: {
      image: [
        { value: "cirros", label: "cirros" },
        { value: "ubuntu-24.04", label: "ubuntu-24.04" },
      ],
      flavor_small: [{ value: "m1.tiny", label: "m1.tiny (1 vCPU, 512 MB RAM, 1 GB disk)" }],
      external_network: [],
    },
  },
};

describe("settingView", () => {
  it("draws the cloud's list, showing the default when nothing is set", () => {
    expect(settingView(flavor, ready, undefined, false)).toEqual({
      control: "list",
      current: "m1.tiny",
      choices: [{ value: "m1.tiny", label: "m1.tiny (1 vCPU, 512 MB RAM, 1 GB disk)" }],
    });
    const v = settingView(image, ready, "ubuntu-24.04", false);
    expect(v.control === "list" && v.current).toBe("ubuntu-24.04");
  });

  it("keeps a typed value the cloud does not list, first and marked", () => {
    const v = settingView(image, ready, "debian", false);
    expect(v.control).toBe("list");
    if (v.control !== "list") return;
    expect(v.current).toBe("debian");
    expect(v.choices[0]).toEqual({ value: "debian", label: "debian (typed name)" });
    expect(v.choices).toHaveLength(3);
  });

  it("lets the user type instead, with a way back to the list", () => {
    expect(settingView(image, ready, "cir", true)).toEqual({ control: "field", note: null, loading: false, canChoose: true });
  });

  it("falls back to a text box and says why", () => {
    expect(settingView(network, ready, undefined, false)).toMatchObject({ control: "field", note: expect.stringContaining("lists none") });
    const noList: OptionsState = { status: "ready", options: { settings: { image: [] } } };
    expect(settingView(flavor, noList, undefined, false)).toMatchObject({ control: "field", note: expect.stringContaining("gave no list") });
    const failed: OptionsState = { status: "error", message: "The cloud refused this application credential.", unsupported: false };
    expect(settingView(image, failed, "cirros", false)).toMatchObject({
      control: "field",
      note: "Could not list choices from your cloud: The cloud refused this application credential. Type the name instead.",
    });
  });

  it("shows loading, and says nothing for a cloud without lists", () => {
    expect(settingView(image, { status: "loading" }, undefined, false)).toMatchObject({ control: "field", loading: true });
    expect(settingView(image, { status: "error", message: "", unsupported: true }, undefined, false)).toEqual({
      control: "field",
      note: null,
      loading: false,
      canChoose: false,
    });
  });

  it("leaves enum and other typed settings to the manifest's input", () => {
    expect(settingView(size, ready, undefined, false)).toEqual({ control: "field", note: null, loading: false, canChoose: false });
  });
});

describe("applySuggestions", () => {
  const ubuntu = "Ubuntu 24.04.2 LTS (x86_64) [2025-04-03]";
  // A public cloud without DevStack's defaults (VEXXHOST-like), as B answers.
  const publicCloud: ConnectionOptions = {
    settings: {
      image: [
        { value: ubuntu, label: ubuntu },
        { value: "debian-12-generic-amd64-20241201", label: "debian-12-generic-amd64-20241201" },
      ],
      flavor_small: [
        { value: "v1-standard-1", label: "v1-standard-1 (1 vCPU, 1 GB RAM, 20 GB disk)" },
        { value: "v3-starter-1", label: "v3-starter-1 (1 vCPU, 2 GB RAM, 10 GB disk)" },
      ],
      external_network: [{ value: "public", label: "public" }],
    },
    suggested: { image: ubuntu, flavor_small: "v1-standard-1", external_network: "public" },
  };
  const spec = [image, flavor, network, size];

  it("replaces defaults the cloud lacks, and says which", () => {
    const got = applySuggestions(spec, { image: "cirros", flavor_small: "m1.tiny", external_network: "public" }, publicCloud);
    expect(got.values).toEqual({ image: ubuntu, flavor_small: "v1-standard-1", external_network: "public" });
    expect(got.picked).toEqual(["image", "flavor_small"]);
  });

  it("fills a setting that is not set yet", () => {
    expect(applySuggestions([image], {}, publicCloud)).toEqual({ values: { image: ubuntu }, picked: ["image"] });
  });

  it("keeps a value the user chose", () => {
    const got = applySuggestions(spec, { image: "debian-12-generic-amd64-20241201", flavor_small: "v3-starter-1" }, publicCloud);
    expect(got.values).toMatchObject({ image: "debian-12-generic-amd64-20241201", flavor_small: "v3-starter-1" });
    expect(got.picked).toEqual([]);
    // A typed value not in the list is the user's too.
    expect(applySuggestions([image], { image: "my-image" }, publicCloud).picked).toEqual([]);
  });

  it("keeps a default the cloud lists", () => {
    const devstack: ConnectionOptions = {
      settings: { flavor_small: [{ value: "m1.tiny", label: "m1.tiny" }] },
      suggested: { flavor_small: "m1.tiny" },
    };
    expect(applySuggestions([flavor], { flavor_small: "m1.tiny" }, devstack).picked).toEqual([]);
  });

  it("ignores a suggestion that is not one of the choices, or with no list", () => {
    const odd: ConnectionOptions = { settings: { image: [{ value: ubuntu, label: ubuntu }] }, suggested: { image: "other", flavor_small: "v1-standard-1" } };
    expect(applySuggestions(spec, { image: "cirros", flavor_small: "m1.tiny" }, odd)).toEqual({
      values: { image: "cirros", flavor_small: "m1.tiny" },
      picked: [],
    });
    expect(applySuggestions(spec, { image: "cirros" }, { settings: null }).picked).toEqual([]);
  });

  it("leaves enum settings alone", () => {
    const sizes: ConnectionOptions = { settings: { size: [{ value: "large", label: "large" }] }, suggested: { size: "large" } };
    expect(applySuggestions([size], {}, sizes).picked).toEqual([]);
  });
});

describe("optionsState", () => {
  it("maps the query", () => {
    expect(optionsState({ isLoading: true, isError: false, errorMessage: "" })).toEqual({ status: "loading" });
    expect(optionsState({ isLoading: false, isError: true, errorCode: "no_options", errorMessage: "x" })).toEqual({
      status: "error",
      message: "x",
      unsupported: true,
    });
    expect(optionsState({ isLoading: false, isError: true, errorCode: "options_failed", errorMessage: "slow" })).toEqual({
      status: "error",
      message: "slow",
      unsupported: false,
    });
    expect(optionsState({ isLoading: false, isError: false, errorMessage: "" })).toMatchObject({ status: "error", unsupported: true });
    expect(optionsState({ isLoading: false, isError: false, data: { settings: {} }, errorMessage: "" })).toEqual({
      status: "ready",
      options: { settings: {} },
    });
  });
});
