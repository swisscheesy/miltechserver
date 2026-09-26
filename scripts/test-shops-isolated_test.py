"""Run the wrapper's migration and marker shell blocks without PostgreSQL."""
from pathlib import Path
import subprocess
import tempfile
import unittest

WRAPPER = Path(__file__).with_name("test-shops-isolated.sh").read_text()
MIGRATIONS = WRAPPER[WRAPPER.index("for migration in "):WRAPPER.index("psql_local -v marker=")]
MARKER = WRAPPER[WRAPPER.index("stored=$("):WRAPPER.index("export TEST_DATABASE_URL=")]


class IsolatedWrapperTests(unittest.TestCase):
    def run_shell(self, code):
        with tempfile.TemporaryDirectory() as directory:
            env = {"PATH": "/usr/bin:/bin", "TEST_TEMP": directory}
            result = subprocess.run(
                ["/bin/bash", "-c", "set -euo pipefail\numask 077\n"
                 'instance="$TEST_TEMP"\nroot=/fixture\n'
                 "fail() { printf '%s\\n' \"$1\" >&2; exit 1; }\n" + code],
                capture_output=True, text=True, env=env,
            )
            log = Path(directory, "setup.log")
            return result, log.read_text() if log.exists() else ""

    def test_empty_migrations_skip_psql_on_macos_bash(self):
        result, _ = self.run_shell(
            "later_migrations=()\npsql_local() { exit 99; }\n" + MIGRATIONS)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_nonempty_migrations_preserve_order_and_spaces(self):
        result, _ = self.run_shell(
            'later_migrations=("one.sql" "two words.sql")\n'
            'psql_local() { printf "%s\\n" "$2" >> "$instance/calls"; }\n'
            + MIGRATIONS + 'cat "$instance/calls"\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, "/fixture/one.sql\n/fixture/two words.sql\n")

    def test_marker_query_failure_hides_diagnostics(self):
        result, log = self.run_shell(
            'marker=expected\npsql_local() { echo sensitive-diagnostic >&2; return 7; }\n'
            + MARKER)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stderr, "Disposable instance marker verification failed.\n")
        self.assertNotIn("sensitive-diagnostic", result.stdout + result.stderr)
        self.assertIn("sensitive-diagnostic", log)

    def test_marker_mismatch_refuses(self):
        result, _ = self.run_shell(
            'marker=expected\npsql_local() { echo wrong; }\n' + MARKER)
        self.assertEqual(result.returncode, 1)
        self.assertEqual(result.stderr, "Disposable instance marker verification failed.\n")

    def test_matching_marker_succeeds(self):
        result, _ = self.run_shell(
            'marker=expected\npsql_local() { echo expected; }\n' + MARKER)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout + result.stderr, "")


if __name__ == "__main__":
    unittest.main(verbosity=2)
