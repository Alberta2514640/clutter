#!/usr/bin/env python3
"""Compare A's test results at the baseline revision with the current tree.

usage: compare.py BASELINE_REV BASELINE.json CURRENT.json OUT.json [RENAMES.tsv]

Matches tests by package and name. Exits 1 when a test that passed at the
baseline fails, or is missing, now.
"""
import json
import sys


def load(path):
    with open(path, encoding="utf-8") as f:
        s = json.load(f)
    return {(t["package"], t["name"]): t["result"] for t in s["tests"]}, s.get("failed_packages", [])


def main() -> int:
    rev, base_path, cur_path, out_path = sys.argv[1:5]
    base, base_bad = load(base_path)
    cur, cur_bad = load(cur_path)
    short = lambda k: f"{k[0].split('/backend/api/')[-1]} {k[1]}"
    by_short = {short(k): r for k, r in cur.items()}
    renames = {}
    if len(sys.argv) > 5:
        for line in open(sys.argv[5], encoding="utf-8"):
            if line.startswith("#") or not line.strip():
                continue
            old, new, reason = line.rstrip("\n").split("\t")
            renames[old] = ([n.strip() for n in new.split(",")], reason)
    renamed = []
    kept, regressed, fixed, new, removed, still_failing = [], [], [], [], [], []
    for k, r in sorted(base.items()):
        now = cur.get(k)
        name = short(k)
        if now is None and name in renames:
            news, reason = renames[name]
            results = {n: by_short.get(n, "missing") for n in news}
            ok = all(v == "pass" for v in results.values())
            renamed.append({"test": name, "baseline": r, "now_covered_by": results, "reason": reason, "ok": ok or r != "pass"})
            if r == "pass" and not ok:
                regressed.append({"test": name, "baseline": r, "current": f"renamed, and {results}"})
            continue
        if now is None:
            removed.append({"test": name, "baseline": r})
            if r == "pass":
                regressed.append({"test": name, "baseline": r, "current": "missing"})
        elif r == "pass" and now == "fail":
            regressed.append({"test": name, "baseline": r, "current": now})
        elif r == "pass" and now == "pass":
            kept.append(name)
        elif r == "fail" and now == "pass":
            fixed.append(name)
        elif r == "fail":
            still_failing.append(name)
    for k, r in sorted(cur.items()):
        if k not in base:
            new.append({"test": f"{k[0].split('/backend/api/')[-1]} {k[1]}", "current": r})
    new_failing = [n for n in new if n["current"] == "fail"]
    out = {
        "baseline": rev,
        "baseline_counts": {r: sum(1 for v in base.values() if v == r) for r in ("pass", "fail", "skip")},
        "current_counts": {r: sum(1 for v in cur.values() if v == r) for r in ("pass", "fail", "skip")},
        "baseline_failed_packages": [p.split("/backend/api/")[-1] for p in base_bad],
        "current_failed_packages": [p.split("/backend/api/")[-1] for p in cur_bad],
        "passing_in_both": len(kept),
        "regressions": regressed,
        "fixed_since_baseline": fixed,
        "still_failing": still_failing,
        "new_tests": len(new),
        "new_tests_failing": new_failing,
        "removed_or_renamed": removed,
        "renamed": renamed,
    }
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump(out, f, indent=1)
    print(f"aws-regression: baseline {out['baseline_counts']}, current {out['current_counts']}; "
          f"{len(kept)} pass in both, {len(regressed)} regressions, {len(fixed)} fixed, "
          f"{len(new)} new ({len(new_failing)} failing), {len(renamed)} renamed (reviewed), {len(removed)} removed")
    for r in regressed:
        print(f"  REGRESSION {r['test']}: {r['baseline']} -> {r['current']}")
    return 1 if regressed or new_failing or cur_bad else 0


if __name__ == "__main__":
    sys.exit(main())
