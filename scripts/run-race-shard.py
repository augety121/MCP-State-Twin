#!/usr/bin/env python3
"""Partition a complete Go test inventory without weakening race or deadlines."""
import argparse
import re
import subprocess


PROFILES = ("2025-11-25", "2026-07-28")
PROFILE_TEST = "TestPluginIndependentProcessProfilesAll24"


def inventory(package):
    listed = subprocess.run(["go", "test", "-list", "^(Test|Fuzz|Example)", package],
                            check=True, capture_output=True, text=True).stdout
    names = sorted(set(line for line in listed.splitlines()
                       if re.fullmatch(r"(?:Test|Fuzz|Example)[A-Za-z0-9_]+", line)))
    if not names:
        raise RuntimeError("empty test inventory")
    # This test contains two independently prepared protocol profiles. Each
    # profile still runs all 24 tasks, including subprocess lifecycle checks.
    expanded = [name for name in names if name != PROFILE_TEST]
    if PROFILE_TEST in names:
        expanded.extend(PROFILE_TEST + "/" + profile for profile in PROFILES)
    return sorted(expanded)


def partition(items, count):
    groups = [items[index::count] for index in range(count)]
    flattened = [item for group in groups for item in group]
    if len(flattened) != len(set(flattened)) or sorted(flattened) != sorted(items):
        raise RuntimeError("incomplete or duplicate inventory")
    return groups


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("package", choices=("./cmd/statetwin", "./internal/agenteval"))
    parser.add_argument("index", type=int)
    parser.add_argument("count", type=int)
    parser.add_argument("--list-only", action="store_true")
    args = parser.parse_args()
    if not 1 <= args.count <= 8 or not 0 <= args.index < args.count:
        parser.error("invalid shard")
    items = inventory(args.package)
    selected = partition(items, args.count)[args.index]
    if not selected:
        raise RuntimeError("empty shard")
    print(f"Inventory {len(items)}; shard {args.index}/{args.count}: {selected}", flush=True)
    if args.list_only:
        return
    normal = [name for name in selected if "/" not in name]
    patterns = ["^(" + "|".join(normal) + ")$"] if normal else []
    patterns.extend("^" + name.replace("/", "$/^") + "$"
                    for name in selected if "/" in name)
    for index, pattern in enumerate(patterns):
        subprocess.run(["go", "test", "-race", "-timeout=20m",
                        f"-coverprofile=coverage-{args.index}-{index}.out",
                        "-run", pattern, args.package], check=True)


if __name__ == "__main__":
    main()
