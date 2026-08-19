#!/usr/bin/env python3
"""Reject a production Terraform plan that crosses project or safety boundaries."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


SAFE_ACTIONS = {"create", "no-op", "read"}


def validate_plan(plan: dict[str, Any], expected_project: str) -> dict[str, int]:
    failures: list[str] = []
    variables = plan.get("variables", {})
    planned_project = variables.get("project_id", {}).get("value")
    if planned_project != expected_project:
        failures.append(
            f"plan project_id is {planned_project!r}; expected {expected_project!r}"
        )

    counts = {"create": 0, "no-op": 0, "read": 0}
    for resource in plan.get("resource_changes", []):
        address = str(resource.get("address", "<unknown>"))
        if not address.startswith("module.platform."):
            failures.append(f"resource is outside module.platform: {address}")

        change = resource.get("change", {})
        actions = set(change.get("actions", []))
        unsafe_actions = actions - SAFE_ACTIONS
        if unsafe_actions:
            failures.append(
                f"resource {address} has forbidden actions: {sorted(actions)}"
            )
        for action in actions & SAFE_ACTIONS:
            counts[action] += 1

        after = change.get("after")
        if isinstance(after, dict):
            resource_project = after.get("project")
            if isinstance(resource_project, str) and resource_project != expected_project:
                failures.append(
                    f"resource {address} targets project {resource_project!r}"
                )

    if failures:
        raise ValueError("\n".join(failures))
    return counts


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--plan", required=True, type=Path)
    parser.add_argument("--expected-project", required=True)
    args = parser.parse_args()

    plan = json.loads(args.plan.read_text(encoding="utf-8"))
    try:
        counts = validate_plan(plan, args.expected_project)
    except ValueError as error:
        print(f"Production apply plan safety failed:\n{error}")
        return 1

    print(
        "Production apply plan safety passed: "
        f"{counts['create']} create, {counts['no-op']} no-op, "
        f"{counts['read']} read; no update, replace, or destroy."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
