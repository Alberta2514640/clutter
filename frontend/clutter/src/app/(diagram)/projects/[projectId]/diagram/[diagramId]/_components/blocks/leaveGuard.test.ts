// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from "vitest";
import { holdLinkClicks, type HeldLink } from "./leaveGuard";

// A link as the editor renders one: its own click handler (React's onClick,
// which chooses the team) runs in the bubble phase, after the guard's
// capture-phase listener.
function link(href: string, attrs: Record<string, string> = {}) {
  const a = document.createElement("a");
  a.href = href;
  for (const [k, v] of Object.entries(attrs)) a.setAttribute(k, v);
  const own = vi.fn((e: MouseEvent) => e.preventDefault());
  a.addEventListener("click", own);
  document.body.append(a);
  return { a, own };
}

afterEach(() => {
  document.body.replaceChildren();
});

describe("holdLinkClicks", () => {
  it("holds a click on a link inside the app before the link's own handler runs", () => {
    const held: HeldLink[] = [];
    const guard = holdLinkClicks(document, (h) => held.push(h));
    const { a, own } = link("/dashboard?x=1#top");
    a.click();
    expect(own).not.toHaveBeenCalled();
    expect(held.map((h) => h.href)).toEqual(["/dashboard?x=1#top"]);
    expect(held[0].link).toBe(a);
    guard.dispose();
  });

  it("follows a held link by running its own handler once, and holds the next click again", () => {
    const held: HeldLink[] = [];
    const guard = holdLinkClicks(document, (h) => held.push(h));
    const { a, own } = link("/settings/organization/connections");
    a.click();
    expect(guard.follow(held[0])).toBe(true);
    expect(own).toHaveBeenCalledTimes(1);
    expect(held).toHaveLength(1);

    a.click();
    expect(own).toHaveBeenCalledTimes(1);
    expect(held).toHaveLength(2);
    guard.dispose();
  });

  it("answers false for a link that is gone, so the caller goes to its href", () => {
    const held: HeldLink[] = [];
    const guard = holdLinkClicks(document, (h) => held.push(h));
    const { a } = link("/dashboard");
    a.click();
    a.remove();
    expect(guard.follow(held[0])).toBe(false);
    expect(guard.follow({ href: "/dashboard", link: null })).toBe(false);
    guard.dispose();
  });

  it("leaves new tabs, downloads, other sites, this page and modified clicks to the browser", () => {
    const onHold = vi.fn();
    const guard = holdLinkClicks(document, onHold);
    link("/dashboard", { target: "_blank" }).a.click();
    link("/file.txt", { download: "" }).a.click();
    link("https://example.org/elsewhere").a.click();
    link(window.location.pathname).a.click();
    link("/dashboard").a.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, metaKey: true }));
    expect(onHold).not.toHaveBeenCalled();
    guard.dispose();
  });

  it("stops holding once disposed", () => {
    const onHold = vi.fn();
    holdLinkClicks(document, onHold).dispose();
    const { a, own } = link("/dashboard");
    a.click();
    expect(onHold).not.toHaveBeenCalled();
    expect(own).toHaveBeenCalledTimes(1);
  });
});
