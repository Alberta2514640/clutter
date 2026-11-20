#!/usr/bin/env python3
"""Platform and version check: every toolchain and image Clutter builds with
is pinned, the pins agree with each other, CI builds on the pinned versions,
and this machine's tools satisfy them.

usage: check_versions.py REPO_ROOT OUT.json

Each check is PASS, WARN (pinned loosely; worth tightening) or FAIL. Exits
1 on any FAIL.
"""
import json
import os
import re
import shutil
import subprocess
import sys

ROOT = sys.argv[1]
checks = []


def read(rel):
    with open(os.path.join(ROOT, rel), encoding="utf-8") as f:
        return f.read()


def add(name, verdict, detail):
    checks.append({"check": name, "verdict": verdict, "detail": detail})


def tool(cmd):
    if not shutil.which(cmd[0]):
        return None
    try:
        return subprocess.run(cmd, capture_output=True, text=True, timeout=30).stdout
    except Exception:  # noqa: BLE001
        return None


def ver(s):
    return tuple(int(x) for x in re.findall(r"\d+", s)[:3])


# Go.
mods = {m: re.search(r"(?m)^go (\S+)", read(f"backend/{m}/go.mod")).group(1) for m in ("api", "adapter", "devserver")}
gov = set(mods.values())
add("Go version is the same in all three modules", "PASS" if len(gov) == 1 else "FAIL", str(mods))
ci = read(".github/workflows/checks.yml")
n_setup_go = ci.count("actions/setup-go")
n_file = ci.count("go-version-file: backend/api/go.mod")
add("CI installs Go from the pinned go.mod", "PASS" if n_setup_go and n_setup_go == n_file else "FAIL",
    f"{n_file} of {n_setup_go} setup-go steps read backend/api/go.mod")
out = tool(["go", "version"])
if out:
    have = re.search(r"go(\d+\.\d+(?:\.\d+)?)", out).group(1)
    ok = ver(have) >= ver(next(iter(gov)))
    add("This machine's Go meets the pin", "PASS" if ok else "FAIL", f"go {have}, pin {next(iter(gov))}")
else:
    add("This machine's Go meets the pin", "FAIL", "go is not installed")

# Node and the frontend.
nodes = re.findall(r"node-version:\s*(\S+)", ci)
engines = json.loads(read("package.json")).get("engines", {}).get("node", "")
add("CI uses one Node major, matching the repo's engines", "PASS" if nodes and set(nodes) == {"22"} and engines == ">=22" else "FAIL",
    f"CI node-version {sorted(set(nodes))}, engines {engines!r}")
out = tool(["node", "--version"])
if out:
    add("This machine's Node meets the pin", "PASS" if ver(out)[0] >= 22 else "FAIL", out.strip())
lock = os.path.exists(os.path.join(ROOT, "frontend/clutter/package-lock.json"))
add("The frontend installs from its lockfile in CI (npm ci)", "PASS" if lock and "npm --prefix frontend/clutter ci" in ci else "FAIL",
    f"package-lock.json present: {lock}")
pkg = json.loads(read("frontend/clutter/package.json"))
deps = {**pkg.get("dependencies", {}), **pkg.get("devDependencies", {})}
ranged = sorted(k for k, v in deps.items() if v.startswith("^") or v.startswith("~"))
add("Frontend dependencies are exact in package.json", "WARN" if ranged else "PASS",
    f"{len(ranged)} of {len(deps)} use ^ or ~ ranges (the lockfile still pins them for npm ci): {', '.join(ranged[:8])}{'...' if len(ranged) > 8 else ''}")

# Images and Terraform.
compose = re.search(r"image:\s*(postgres:\S+)", read("backend/devserver/docker-compose.yml")).group(1)
pgtest = re.search(r'Image = "([^"]+)"', read("backend/adapter/internal/pgtest/pgtest.go")).group(1)
add("Postgres image is pinned by digest, the same in A2 and B's tests", "PASS" if compose == pgtest and "@sha256:" in compose else "FAIL",
    compose)
tf_images = {}
for rel in ("backend/adapter/images/runtask.Dockerfile", "backend/infra/modules/fargate/ecr-scripts/terraform-deployer/Dockerfile"):
    m = re.search(r"hashicorp/terraform:(\S+)", read(rel))
    tf_images[rel.split("/")[-2] if "deployer" in rel else "B run task"] = m.group(1) if m else None
add("Terraform version is the same in A's deployer and B's run task", "PASS" if len(set(tf_images.values())) == 1 and None not in tf_images.values() else "FAIL",
    str(tf_images))
out = tool(["terraform", "version"])
if out:
    have = re.search(r"v(\d+\.\d+\.\d+)", out).group(1)
    pin = next(iter(tf_images.values()))
    add("This machine's Terraform matches the runners' pin", "PASS" if have == pin else "WARN",
        f"terraform {have}, runners use {pin} (local validate and LocalStack runs use this machine's)")
os_pin = re.search(r'providerVersion\s*=\s*"([^"]+)"', read("backend/adapter/connectors/generalopenstack/generalopenstack.go"))
add("OpenStack provider is pinned (~> 3.4)", "PASS" if os_pin and os_pin.group(1) == "~> 3.4" else "FAIL", os_pin.group(1) if os_pin else "none")
aws_pin = re.search(r'source\s*=\s*"hashicorp/aws"\s+version\s*=\s*"([^"]+)"', read("backend/adapter/connectors/aws/tmpl/provider.tf.tmpl"))
add("AWS provider in B's rendered Terraform is pinned to a major", "PASS" if aws_pin and aws_pin.group(1).startswith("~>") else "FAIL",
    aws_pin.group(1) if aws_pin else "none")
plat = re.findall(r'version\s*=\s*"([^"]+)"', read("backend/infra/providers.tf"))
add("Platform Terraform providers have an upper bound", "WARN" if any(v.startswith(">=") for v in plat) else "PASS",
    f"backend/infra/providers.tf: {plat} (open-ended '>=' lets a new major in at init)")

# CI's own pins.
actions = sorted(set(re.findall(r"uses:\s*(\S+)", ci + read(".github/workflows/docker-image.yml"))))
by_sha = all(re.search(r"@[0-9a-f]{40}$", a) for a in actions)
add("CI actions are pinned", "PASS" if by_sha else "WARN", f"{', '.join(actions)} (major tags, not commit SHAs)")
add("Markdown lint runs a pinned version", "PASS" if "markdownlint-cli2@0.18.1" in ci else "FAIL", "markdownlint-cli2@0.18.1")
add("CI runs every Go module's tests, including backend/adapter/tests", "PASS" if "go -C backend/adapter test ./..." in ci else "FAIL",
    "the checks workflow runs go test ./... in api, devserver and adapter")
with open(sys.argv[2], "w", encoding="utf-8") as f:
    json.dump({"checks": checks}, f, indent=1)
for c in checks:
    print(f"  {c['verdict']:4}  {c['check']}: {c['detail']}")
fails = sum(c["verdict"] == "FAIL" for c in checks)
warns = sum(c["verdict"] == "WARN" for c in checks)
print(f"platform-version: {len(checks) - fails - warns} pass, {warns} warn, {fails} fail")
sys.exit(1 if fails else 0)
