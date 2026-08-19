#!/usr/bin/env python3
"""Unit tests for the production Terraform plan safety gate."""

import unittest

from check_production_apply_plan import validate_plan


PROJECT = "rentstage-prod-example"


def plan_for(actions: list[str], *, project: str = PROJECT) -> dict:
    return {
        "variables": {"project_id": {"value": PROJECT}},
        "resource_changes": [
            {
                "address": "module.platform.google_sql_database.rentstage",
                "change": {
                    "actions": actions,
                    "after": {"project": project},
                },
            }
        ],
    }


class ValidateProductionPlanTests(unittest.TestCase):
    def test_accepts_create_only_plan(self) -> None:
        self.assertEqual(
            validate_plan(plan_for(["create"]), PROJECT),
            {"create": 1, "no-op": 0, "read": 0},
        )

    def test_rejects_update(self) -> None:
        with self.assertRaisesRegex(ValueError, "forbidden actions"):
            validate_plan(plan_for(["update"]), PROJECT)

    def test_rejects_replacement_or_destroy(self) -> None:
        with self.assertRaisesRegex(ValueError, "forbidden actions"):
            validate_plan(plan_for(["delete", "create"]), PROJECT)

    def test_rejects_wrong_input_project(self) -> None:
        plan = plan_for(["create"])
        plan["variables"]["project_id"]["value"] = "staging-project"
        with self.assertRaisesRegex(ValueError, "plan project_id"):
            validate_plan(plan, PROJECT)

    def test_rejects_cross_project_resource(self) -> None:
        with self.assertRaisesRegex(ValueError, "targets project"):
            validate_plan(plan_for(["create"], project="staging-project"), PROJECT)

    def test_rejects_root_resource(self) -> None:
        plan = plan_for(["create"])
        plan["resource_changes"][0]["address"] = "google_sql_database.rentstage"
        with self.assertRaisesRegex(ValueError, "outside module.platform"):
            validate_plan(plan, PROJECT)


if __name__ == "__main__":
    unittest.main()
