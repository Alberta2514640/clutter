#!/usr/bin/env python3
"""Summarize `go test -json` output into one JSON file for the test report.

usage: gotest_summary.py SUITE_NAME INPUT.jsonl OUTPUT.json

Writes every test's result (pass, fail or skip), its time, and the output
lines it logged, and prints one line of counts. Exits 1 when a test or a
package failed, so a run script can stop on it.
"""
import json
import sys


def main() -> int:
    if len(sys.argv) != 4:
        print(__doc__, file=sys.stderr)
        return 2
    suite, src, dst = sys.argv[1:]
    tests = {}
    packages = {}
    with open(src, encoding="utf-8", errors="replace") as f:
        for line in f:
            line = line.strip()
            if not line.startswith("{"):
                continue
            try:
                ev = json.loads(line)
            except json.JSONDecodeError:
                continue
            pkg = ev.get("Package", "")
            name = ev.get("Test")
            action = ev.get("Action")
            if not name:
                if action in ("pass", "fail", "skip"):
                    packages[pkg] = action
                if action == "output" and "build failed" in ev.get("Output", ""):
                    packages[pkg] = "fail"
                continue
            key = (pkg, name)
            t = tests.setdefault(key, {"package": pkg, "name": name, "result": "run", "seconds": 0.0, "output": []})
            if action in ("pass", "fail", "skip"):
                t["result"] = action
                t["seconds"] = ev.get("Elapsed", 0.0)
            elif action == "output":
                out = ev.get("Output", "")
                if not out.startswith("=== ") and not out.startswith("--- ") and out.strip() not in ("PASS", "FAIL"):
                    if len(t["output"]) < 60:
                        t["output"].append(out.rstrip("\n"))
    rows = sorted(tests.values(), key=lambda t: (t["package"], t["name"]))
    counts = {"pass": 0, "fail": 0, "skip": 0, "run": 0}
    for t in rows:
        counts[t["result"]] = counts.get(t["result"], 0) + 1
    failed_pkgs = [p for p, a in packages.items() if a == "fail"]
    summary = {"suite": suite, "counts": counts, "packages": packages, "failed_packages": failed_pkgs, "tests": rows}
    with open(dst, "w", encoding="utf-8") as f:
        json.dump(summary, f, indent=1)
    top = [t for t in rows if "/" not in t["name"]]
    top_counts = {r: sum(1 for t in top if t["result"] == r) for r in ("pass", "fail", "skip")}
    print(f"{suite}: {counts['pass']} pass, {counts['fail']} fail, {counts['skip']} skip "
          f"(top-level tests: {top_counts['pass']} pass, {top_counts['fail']} fail, {top_counts['skip']} skip)")
    if not rows:
        print(f"{suite}: no test ran; see the .stderr file beside the results", file=sys.stderr)
        return 1
    return 1 if counts["fail"] or failed_pkgs else 0


if __name__ == "__main__":
    sys.exit(main())
