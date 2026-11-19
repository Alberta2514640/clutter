#!/usr/bin/env python3
"""Check the customer role template (phase 0 item 7) for wildcard grants.

usage: check_role.py ROLE.yaml

Fails on any Allow statement whose Action is "*" or a whole service
("iam:*", "s3:*", "ec2:*", "ssm:*", and so on). Deny statements may use
wildcards: an explicit deny only narrows. A line-based reader, because the
template's CloudFormation tags (!Sub) are not plain YAML.
"""
import re
import sys


def statements(lines):
    cur = None
    for raw in lines:
        line = raw.split("#", 1)[0].rstrip()
        if re.match(r"^\s*-\s+(Sid|Effect):", line):
            if cur:
                yield cur
            cur = {"effect": None, "actions": [], "start": raw}
        if cur is None:
            continue
        m = re.search(r"\bEffect:\s*(\w+)", line)
        if m:
            cur["effect"] = m.group(1)
        m = re.search(r"\b(?:Not)?Action:\s*(\S.*)?$", line)
        if m and m.group(1):
            cur["actions"].append(m.group(1).strip().strip('"'))
        m = re.match(r"^\s*-\s+\"?([a-z0-9-]+:[A-Za-z0-9*]+|\*)\"?\s*$", line)
        if m:
            cur["actions"].append(m.group(1))
    if cur:
        yield cur


def main() -> int:
    path = sys.argv[1]
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()
    bad, allows = [], 0
    for st in statements(lines):
        if st["effect"] != "Allow":
            continue
        allows += 1
        for a in st["actions"]:
            if a == "*" or a.endswith(":*"):
                bad.append(a)
    print(f"customer role: {allows} Allow statements checked; {len(bad)} wildcard actions {bad}")
    return 1 if bad or allows == 0 else 0


if __name__ == "__main__":
    sys.exit(main())
