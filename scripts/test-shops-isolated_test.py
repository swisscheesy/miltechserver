"""Execute wrapper refusals/cleanup with external PostgreSQL and Go tools replaced."""
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]
FAKE_TOOL = r'''#!/usr/bin/env python3
import json, os, pathlib, sys
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
base = pathlib.Path(os.environ["HARNESS_TEMP"])
with (base / "calls").open("a") as log:
    log.write(json.dumps([name, args, os.getcwd()]) + "\n")
def value(flag):
    return args[args.index(flag) + 1]
if name == "mktemp":
    target = base / "cluster"
    target.mkdir()
    print(target)
elif name == "initdb":
    pathlib.Path(value("-D")).mkdir()
    sys.exit(int(os.environ.get("FAIL_INITDB", "0")))
elif name == "pg_ctl":
    pass
elif name == "psql":
    if "-c" in args and "pg_trgm" in value("-c"):
        pass
    elif "-Atc" in args:
        query = value("-Atc")
        if "extnamespace" in query:
            print("public")
        elif "disposable_instance" in query:
            marker = (base / "marker").read_text()
            if os.environ.get("BAD_MARKER"):
                marker = ""
            if "current_database()" in query:
                print(value("-d") + "|postgres|" + marker)
            else:
                print(marker)
    else:
        for arg in args:
            if arg.startswith("marker="):
                (base / "marker").write_text(arg[7:])
        if "-f" not in args:
            sys.stdin.read()
elif name == "go":
    sys.exit(71)
'''


class IsolatedWrapperTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.base = Path(self.temp.name)
        self.bin = self.base / "bin"
        self.bin.mkdir()
        for name in ("mktemp", "initdb", "pg_ctl", "psql", "createdb", "go"):
            tool = self.bin / name
            tool.write_text(FAKE_TOOL)
            tool.chmod(0o700)
        # Bind selection is external to the safety logic and prohibited in some sandboxes.
        python = self.bin / "python3"
        python.write_text("#!/bin/bash\nif [[ ${2-} == *'import socket'* ]]; then echo 55432; else exec "
                          + shutil.which("python3") + ' "$@"; fi\n')
        python.chmod(0o700)
        self.env = {"PATH": str(self.bin) + ":" + os.environ["PATH"],
                    "HOME": str(self.base), "HARNESS_TEMP": str(self.base)}

    def run_wrapper(self, arguments=None, extra_env=None):
        env = self.env | (extra_env or {})
        result = subprocess.run(["/bin/bash", str(ROOT / "scripts/test-shops-isolated.sh")]
                                + (arguments or ["./tests/shops"]),
                                env=env, capture_output=True, text=True)
        log = self.base / "calls"
        calls = [json.loads(line) for line in log.read_text().splitlines()] if log.exists() else []
        return result, calls

    def assert_no_target_contact(self, result, calls):
        self.assertNotEqual(result.returncode, 0)
        existing_database_connection_attempts = len([c for c in calls if c[0] in ("psql", "createdb")])
        self.assertEqual(existing_database_connection_attempts, 0)
        self.assertFalse((self.base / "cluster").exists())

    def test_inherited_target_refused(self):
        for name in ("TEST_DATABASE_URL", "TEST_DATABASE_MARKER", "TEST_DB_URL"):
            with self.subTest(name=name):
                result, calls = self.run_wrapper(extra_env={name: ""})
                self.assert_no_target_contact(result, calls)

    def test_unsupported_package_refused(self):
        for package in ("./...", "./tests/...", "./tests/user_pmcs", "./tests/shops/..."):
            with self.subTest(package=package):
                result, calls = self.run_wrapper([package])
                self.assert_no_target_contact(result, calls)

    def test_current_psql_arguments(self):
        result, calls = self.run_wrapper()
        self.assertNotEqual(result.returncode, 0)  # Deliberate Go failure.
        psql = [args for name, args, _ in calls if name == "psql"]
        self.assertTrue(psql, result.stderr)
        for args in psql:
            self.assertEqual(args[:10], ["-X", "-v", "ON_ERROR_STOP=1", "-h",
                             str(self.base / "cluster"), "-p", "55432", "-U", "postgres", "-d"])
            self.assertTrue(args[10].startswith("miltech_test_"))
        self.assertFalse((self.base / "cluster").exists())

    def test_marker_required(self):
        result, calls = self.run_wrapper(extra_env={"BAD_MARKER": "1"})
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("marker", result.stderr.lower())
        self.assertFalse(any(name == "go" for name, _, _ in calls))
        self.assertFalse((self.base / "cluster").exists())

    def test_cleanup_on_failure(self):
        result, calls = self.run_wrapper(extra_env={"FAIL_INITDB": "8"})
        self.assert_no_target_contact(result, calls)
        self.assertTrue(any(name == "pg_ctl" and "stop" in args for name, args, _ in calls))

    def test_remediation_flag_reaches_guarded_setup(self):
        result, calls = self.run_wrapper(["--verify-remediation-migrations", "./tests/shops"])
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any(name == "createdb" for name, _, _ in calls), result.stderr)
        self.assertFalse((self.base / "cluster").exists())

    def test_release_runner_flag_preserves_guarded_setup(self):
        result, calls = self.run_wrapper(["--verify-release-runner", "./tests/shops"])
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(any(name == "createdb" for name, _, _ in calls), result.stderr)
        self.assertFalse((self.base / "cluster").exists())

    def test_helper_refuses_standalone_target(self):
        result = subprocess.run(["/bin/bash", str(ROOT / "scripts/verify-shops-remediation-migrations.sh"),
                                 "postgres://example.invalid/miltech_ng"],
                                env=self.env, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("wrapper", result.stderr.lower())
        self.assertFalse((self.base / "calls").exists())

    def run_helper(self, code, root):
        return subprocess.run(["/bin/bash", "-c",
            'set -euo pipefail\nsource "$HELPER"\n'
            'fail() { echo "$1" >&2; exit 1; }\n' + code],
            env=self.env | {"HELPER": str(ROOT / "scripts/verify-shops-remediation-migrations.sh"),
                            "root": str(root), "instance": str(self.base)},
            capture_output=True, text=True)

    def test_migration_pairs_preserve_order_and_refuse_gaps(self):
        root = self.base / "fixture with spaces"
        migrations = root / "migrations"
        migrations.mkdir(parents=True)
        pairs = [
            ("019_fix_shop_message_allocator_lock_order.sql", "019_rollback_fix_shop_message_allocator_lock_order.sql"),
            ("020_create_shop_message_asset_lifecycle.sql", "020_rollback_shop_message_asset_lifecycle.sql"),
        ]
        for forward, reverse in reversed(pairs):
            (migrations / forward).touch()
            (migrations / reverse).touch()
        code = 'collect_remediation_migrations\nprintf "%s\\n" "${remediation_migrations[@]}"'
        result = self.run_helper(code, root)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout.splitlines(), [
            "migrations/019_fix_shop_message_allocator_lock_order.sql",
            "migrations/020_create_shop_message_asset_lifecycle.sql"])
        (migrations / pairs[0][0]).unlink()
        result = self.run_helper(code, root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("incomplete", result.stderr)
        (migrations / pairs[0][1]).unlink()
        result = self.run_helper(code, root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("ordered prefix", result.stderr)
        (migrations / pairs[0][0]).symlink_to(root / "missing.sql")
        result = self.run_helper(code, root)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("incomplete", result.stderr)
        self.assertFalse((self.base / "calls").exists())

    def test_each_forward_regenerates_before_next_migration(self):
        root = self.base / "candidate"
        root.mkdir()
        result = self.run_helper(r'''marker=$(printf '%064d' 0)
workspace="$root"
remediation_migrations=(migrations/019.sql migrations/020.sql)
psql_local() {
  if [[ $2 == -Atc ]]; then printf '%s|postgres|%s\n' "$1" "$marker";
  else printf 'apply %s\n' "${3##*/}" >> "$instance/events"; fi
}
regenerate_stage() { printf 'generate %s\n' "$2" >> "$instance/events"; }
apply_remediation_migrations miltech_test_shops
''', root)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.base / "events").read_text().splitlines(), [
            "apply 019.sql", "generate 019.sql", "apply 020.sql", "generate 020.sql"])

    def test_candidate_copy_excludes_prior_models_and_configuration(self):
        root = self.base / "candidate"
        names = ["go.mod", "go.sum", "main.go", "helper/current.go", "api/new.go",
                 ".env", "fire_auth_key.json", ".gen/miltech_ng/public/model/old.go",
                 ".superpowers/cache.go"]
        for name in names:
            file = root / name
            file.parent.mkdir(parents=True, exist_ok=True)
            file.write_text("current candidate " + name)
        # Snapshotting imports the shared candidate selector from the candidate
        # tree, just as it does in the real worktree.
        collector = root / "scripts/shops_candidate_inputs.py"
        collector.parent.mkdir(parents=True)
        collector.write_bytes((ROOT / "scripts/shops_candidate_inputs.py").read_bytes())
        git = self.bin / "git"
        git.write_text("#!/usr/bin/env python3\nimport sys\nsys.stdout.write("
                       + repr("\0".join(names) + "\0") + ")\n")
        git.chmod(0o700)
        result = self.run_helper('build_go() { return 0; }\nprepare_generation_workspace', root)
        self.assertEqual(result.returncode, 0, result.stderr)
        output = self.base / "source"
        self.assertEqual((output / "helper/current.go").read_text(), "current candidate helper/current.go")
        self.assertTrue((output / "api/new.go").is_file())
        for name in (".gen", ".env", "fire_auth_key.json", ".superpowers"):
            self.assertFalse((output / name).exists())


if __name__ == "__main__":
    unittest.main(verbosity=2)
