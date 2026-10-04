"""Refusal-only checks; successful publication is exercised by the guarded wrapper."""
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

from shops_candidate_inputs import collect_candidate_inputs

ROOT = Path(__file__).resolve().parent.parent
SCRIPT = ROOT / 'scripts/publish-shops-candidate-generated.py'

class PublicationRefusals(unittest.TestCase):
    def test_refusals_preserve_current_generated_input(self):
        tracked = subprocess.check_output(['git', 'ls-files', '.gen'], cwd=ROOT, text=True).splitlines()
        before = {name: hashlib.sha256((ROOT/name).read_bytes()).hexdigest() for name in tracked}
        with tempfile.TemporaryDirectory() as temporary:
            work = Path(temporary)
            candidate = work/'source'
            candidate.mkdir()
            reference = work/'reference.json'
            reference.write_text(json.dumps(before))
            provenance = work/'source.json'
            provenance.write_text('{}')
            def refuse(root, expected):
                result = subprocess.run(['python3', str(SCRIPT), str(root), str(candidate), str(reference), str(provenance)],capture_output=True,text=True)
                self.assertNotEqual(result.returncode,0)
                self.assertIn(expected,result.stderr)
                self.assertEqual(before,{name: hashlib.sha256((ROOT/name).read_bytes()).hexdigest() for name in tracked})
            refuse(work,'approved linked worktree')
            (candidate/'.gen').symlink_to(ROOT/'.gen',target_is_directory=True)
            refuse(ROOT,'symlinked generated entries')
            (candidate/'.gen').unlink()
            (candidate/'.gen').mkdir()
            reference.write_text('{}')
            refuse(ROOT,'32 tracked PMCS references')
            reference.write_text(json.dumps(before))
            for name in tracked:
                path=candidate/name;path.parent.mkdir(parents=True,exist_ok=True);path.write_bytes((ROOT/name).read_bytes())
            changed=candidate/tracked[0];changed.write_text('incorrect generated source')
            refuse(ROOT,'tracked PMCS equality failed')
            changed.write_bytes((ROOT/tracked[0]).read_bytes())
            manifest=candidate/'.gen/miltech_ng/public/generation-manifest.json'
            manifest.write_text(json.dumps({'database':'miltech_test_asset_lifecycle','user':'postgres','schema':'public'}))
            refuse(ROOT,'full candidate identity')

    def test_new_eligible_input_refuses_before_rename(self):
        self.assert_source_refusal('new-input', 'Publication source input set changed since candidate build')

    def test_changed_eligible_input_refuses_before_rename(self):
        self.assert_source_refusal('changed-input', 'Publication source changed since candidate build')

    def test_redirected_source_directory_refuses_before_rename(self):
        self.assert_source_refusal('redirected-directory', 'Candidate source missing or redirected')

    def assert_source_refusal(self, scenario, expected):
        before = {str(p.relative_to(ROOT/'.gen')): hashlib.sha256(p.read_bytes()).hexdigest()
                  for p in (ROOT/'.gen').rglob('*') if p.is_file()}
        with tempfile.TemporaryDirectory() as temporary, tempfile.TemporaryDirectory(prefix='publication-proof-', dir=ROOT/'api') as fixture:
            work = Path(temporary)
            candidate = work/'source'
            reference = work/'reference.json'
            provenance = work/'source.json'
            shutil.copytree(ROOT/'.gen', candidate/'.gen')
            reference.write_bytes((ROOT/'.gen/pmcs-reference-manifest.json').read_bytes())
            source = {'inputs': collect_candidate_inputs(ROOT)[0]}
            fixture_path = Path(fixture)
            leaf = fixture_path/'new_input.go'
            if scenario in ('redirected-directory', 'changed-input'):
                leaf.write_text('package publicationproof\n')
                source['inputs'][str(leaf.relative_to(ROOT))] = hashlib.sha256(leaf.read_bytes()).hexdigest()
            if scenario == 'redirected-directory':
                redirected = work/'redirected'
                shutil.move(fixture_path, redirected)
                fixture_path.symlink_to(redirected, target_is_directory=True)
            elif scenario == 'changed-input':
                leaf.write_text('package changedpublicationproof\n')
            else:
                leaf.write_text('package publicationproof\n')
            provenance.write_text(json.dumps(source))
            # A broken refusal must never publish the test fixture into real .gen.
            # The real script runs, but any attempted rename fails before mutation.
            driver = """import os, runpy, sys
from unittest.mock import patch
script, *args = sys.argv[1:]
sys.path.insert(0, os.path.dirname(script))
sys.argv = [script, *args]
with patch('os.rename', side_effect=AssertionError('publication reached rename')):
    runpy.run_path(script, run_name='__main__')
"""
            try:
                result = subprocess.run([sys.executable, '-c', driver, str(SCRIPT), str(ROOT), str(candidate), str(reference), str(provenance)], capture_output=True, text=True)
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(expected, result.stderr)
                self.assertNotIn('publication reached rename', result.stderr)
            finally:
                if fixture_path.is_symlink():
                    fixture_path.unlink()
                    fixture_path.mkdir()
                after = {str(p.relative_to(ROOT/'.gen')): hashlib.sha256(p.read_bytes()).hexdigest()
                         for p in (ROOT/'.gen').rglob('*') if p.is_file()}
                self.assertEqual(before, after)

if __name__=='__main__':
    unittest.main()
