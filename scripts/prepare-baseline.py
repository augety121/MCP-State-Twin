"""Prepare a fresh synthetic baseline workspace; never change global config."""
import argparse
import json
from pathlib import Path
import shutil
import subprocess


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--binary", required=True)
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    source = Path(__file__).resolve().parents[1] / "examples"
    output = Path(args.out).resolve()
    binary = Path(args.binary).resolve()
    if output == source or output.is_relative_to(source):
        parser.error("output must be outside examples")
    files = [p for p in source.rglob("*") if p.suffix in (".json", ".yaml")
             and not any(part.startswith(".") for part in p.relative_to(source).parts)]
    if len(files) > 2048 or sum(p.stat().st_size for p in files) > 64 << 20 or any(p.is_symlink() for p in files):
        parser.error("source assets exceed bounds or contain a link")
    output.mkdir(parents=True, exist_ok=False)
    for file in files:
        destination = output / file.relative_to(source)
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(file, destination)
    for domain in ("issue-tracker", "package-registry"):
        root = output / domain
        (root / ".statetwin").mkdir()
        for world in (("agent",) if domain == "issue-tracker" else ("agent", "project")):
            for reviewed in (False, True):
                src = root / "reviewed" / "world" / world if reviewed else root
                name = ("reviewed-" if reviewed else "") + world + "-world.stb"
                subprocess.run([str(binary), "bundle", "build", "--manifest", str(src / f"bundle-{world}.yaml"),
                                "--out", str(root / ".statetwin" / name)], check=True, timeout=120)
    plan = json.loads((output / "plugin-session.json").read_text(encoding="utf-8"))
    plan["host"]["name"], plan["host"]["version"] = "inspect", "0.3.275"
    plan["id"], plan["output"] = "inspect-read-issue", "inspect-read-issue"
    (output / "inspect-session.json").write_text(json.dumps(plan, indent=2), encoding="utf-8")
    subprocess.run([str(binary), "plugin", "check", "--root", str(output), "--pack", "baseline-pack.json",
                    "--profile", "plugin-profile.json"], check=True, timeout=120)


if __name__ == "__main__":
    main()
