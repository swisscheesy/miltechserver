#!/usr/bin/env python3
"""Conservative static checks of our explicit Docker exclusion patterns.

This is not a Docker pattern parser or context/layer/runtime acceptance.
The actual synthetic context and layer sentinel test remains mandatory.
"""
import fnmatch
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
EXCLUDED = ["server.log", "logs/worker.log", "nested/server.log", "nested/logs/worker.log",
            "scratch/probe.txt", "nested/scratch/probe.txt", "tmp/probe.txt", "nested/tmp/probe.txt",
            "temp/probe.txt", "nested/temp/probe.txt", "probe.tmp", "nested/probe.tmp"]
REQUIRED = ["go.mod", "go.sum", "main.go", ".gen/miltech_ng/public/model/shop_messages.go",
            "frontend/package.json", "frontend/package-lock.json", "frontend/src/app.html",
            "bootstrap/env.go", "internal/jetgen/generate.go", "migrations/020_create_shop_message_asset_lifecycle.sql"]

def excluded(path, patterns):
    # Our rules are exclusions only. Check every ancestor, as excluding a
    # directory excludes its children. **/ has zero-or-more directory meaning.
    result = False
    for pattern in patterns.splitlines():
        pattern = pattern.strip()
        if not pattern or pattern.startswith("#"):
            continue
        if pattern.startswith("!"):
            raise AssertionError("negation requires extending this static guard")
        variants = [pattern]
        if pattern.startswith("**/"):
            variants.append(pattern[3:])
        parts = pathlib.PurePosixPath(path).parts
        for size in range(1, len(parts)+1):
            candidate = "/".join(parts[:size])
            if any(fnmatch.fnmatchcase(candidate, variant) for variant in variants):
                result = True
    return result

class ContainerExclusions(unittest.TestCase):
    def test_required_inputs_and_all_log_temp_sentinels(self):
        patterns = (ROOT / ".dockerignore").read_text()
        for path in EXCLUDED:
            with self.subTest(path=path): self.assertTrue(excluded(path, patterns), path)
        for path in REQUIRED:
            with self.subTest(path=path): self.assertFalse(excluded(path, patterns), path)
    def test_guard_rejects_intentionally_missing_log_pattern(self):
        patterns = (ROOT / ".dockerignore").read_text()
        weakened = "\n".join(line for line in patterns.splitlines() if line.strip() != "**/*.log")
        self.assertFalse(excluded("server.log", weakened))
        self.assertFalse(excluded("nested/server.log", weakened))
    def test_runtime_guard_includes_every_sentinel(self):
        guard = (ROOT / "scripts/test-server-container-inputs.sh").read_text()
        for path in EXCLUDED: self.assertIn(repr(path), guard)

if __name__ == "__main__": unittest.main()
