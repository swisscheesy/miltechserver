"""Publish only the verified full disposable candidate into the approved worktree."""
import fcntl
import hashlib
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

from shops_candidate_inputs import collect_candidate_inputs, validate_candidate_source

root, workspace, reference, provenance = map(Path, sys.argv[1:])
expected = Path('/Users/swisscheese/projects/miltechserver/.worktrees/shops-server-release-remediation-20261003')
if root != expected or root.resolve() != expected or not (root / '.git').is_file():
    raise SystemExit('Publication requires the approved linked worktree')
branch = subprocess.check_output(['git', 'branch', '--show-current'], cwd=root, text=True).strip()
if branch != 'codex/shops-server-release-remediation-20261003':
    raise SystemExit('Publication branch identity mismatch')
with (root / '.superpowers/sdd/2026-10-03-shops-server-release-remediation/generated-publication.lock').open('w') as lock:
    fcntl.flock(lock, fcntl.LOCK_EX)
    for tree in (root / '.gen', workspace / '.gen'):
        if tree.is_symlink() or any(p.is_symlink() for p in tree.rglob('*')):
            raise SystemExit('Publication refuses symlinked generated entries')
    hashes = json.loads(reference.read_text())
    if len(hashes) != 32:
        raise SystemExit('Publication requires all 32 tracked PMCS references')
    for name, digest in hashes.items():
        if not name.startswith('.gen/miltech_ng/public/') or not Path(name).name.startswith('user_pmcs_'):
            raise SystemExit('Publication reference path mismatch')
        for base in (root, workspace):
            if hashlib.sha256((base / name).read_bytes()).hexdigest() != digest:
                raise SystemExit('Publication tracked PMCS equality failed')
    if subprocess.check_output(['git', 'diff', 'HEAD', '--', *hashes.keys()], cwd=root):
        raise SystemExit('Publication tracked PMCS files differ from HEAD')
    manifest = json.loads((workspace / '.gen/miltech_ng/public/generation-manifest.json').read_text())
    if manifest['database'] != 'miltech_test_shops' or manifest['user'] != 'postgres' or manifest['schema'] != 'public':
        raise SystemExit('Publication requires full candidate identity')
    source_inputs = json.loads(provenance.read_text()).get('inputs')
    if not source_inputs:
        raise SystemExit('Publication source provenance is missing')
    # Validate recorded paths too: Git no longer enumerates descendants when a
    # directory is replaced with a symlink after the snapshot.
    for name in source_inputs:
        validate_candidate_source(root, Path(name))
    current_inputs, _ = collect_candidate_inputs(root)
    if set(current_inputs) != set(source_inputs):
        raise SystemExit('Publication source input set changed since candidate build')
    if current_inputs != source_inputs:
        raise SystemExit('Publication source changed since candidate build')
    staging = Path(tempfile.mkdtemp(prefix='.candidate-generated-', dir=root))
    backup = staging / 'previous'
    incoming = staging / 'incoming'
    try:
        shutil.copytree(workspace / '.gen', incoming)
        shutil.copyfile(provenance, incoming / 'source-manifest.json')
        shutil.copyfile(reference, incoming / 'pmcs-reference-manifest.json')
        os.rename(root / '.gen', backup)
        try:
            os.rename(incoming, root / '.gen')
        except BaseException:
            os.rename(backup, root / '.gen')
            raise
    finally:
        # If filesystem recovery itself fails, retain the prior complete input.
        if not backup.exists() or (root / '.gen').exists():
            shutil.rmtree(staging)
print('Published untouched canonical full available candidate; all 32 tracked PMCS files unchanged.')
