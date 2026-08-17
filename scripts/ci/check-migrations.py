#!/usr/bin/env python3
from pathlib import Path
import re
import sys
root = Path(__file__).resolve().parents[2]
migrations = root / "apps/api/internal/database/migrations"
files = sorted(p.name for p in migrations.glob("*.sql"))
pattern = re.compile(r"^(\d{3})_[a-z0-9_]+\.sql$")
seen = set()
for name in files:
    match = pattern.match(name)
    if not match:
        raise SystemExit(f"Invalid migration filename: {name}")
    number = int(match.group(1))
    if number in seen:
        raise SystemExit(f"Duplicate migration number: {number:03d}")
    seen.add(number)
expected = list(range(1, max(seen, default=0) + 1))
if sorted(seen) != expected:
    raise SystemExit(f"Migration sequence has gaps: found {sorted(seen)}, expected {expected}")
print(f"Migration ordering passed: {files[0]} through {files[-1]} ({len(files)} files)")
