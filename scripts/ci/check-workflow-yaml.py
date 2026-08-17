#!/usr/bin/env python3
from pathlib import Path
import sys
import yaml
root = Path(__file__).resolve().parents[2]
paths = sorted((root / ".github/workflows").glob("*.yml")) + sorted((root / ".github/workflows").glob("*.yaml"))
paths += [root / "compose.yaml", root / ".github/dependabot.yml"]
for path in paths:
    if not path.exists():
        continue
    try:
        yaml.compose(path.read_text(encoding="utf-8"))
    except Exception as exc:
        raise SystemExit(f"Invalid YAML in {path.relative_to(root)}: {exc}")
print(f"YAML syntax passed for {len(paths)} files.")
