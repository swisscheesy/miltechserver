"""One source selection and path-validation contract for snapshot/publication."""
import hashlib
from pathlib import Path
import subprocess


def validate_candidate_source(root, relative):
    if relative.is_absolute() or '..' in relative.parts:
        raise SystemExit('Candidate source path is invalid')
    source = root / relative
    if not source.is_file() or any(
        (root / Path(*relative.parts[:i])).is_symlink()
        for i in range(1, len(relative.parts) + 1)
    ):
        raise SystemExit('Candidate source missing or redirected')
    return source


def collect_candidate_inputs(root):
    names = subprocess.check_output(
        ['git', 'ls-files', '-z', '--cached', '--others', '--exclude-standard'],
        cwd=root,
    ).decode().split('\0')
    inputs, pmcs = {}, {}
    for name in sorted(set(names) - {''}):
        relative = Path(name)
        selected = (
            name in ('go.mod', 'go.sum')
            or (len(relative.parts) == 1 and relative.suffix == '.go')
            or (relative.parts[0] in ('api', 'bootstrap', 'helper', 'internal', 'tools', 'tests', 'migrations')
                and relative.suffix in ('.go', '.sql'))
            or name == 'api/user_pmcs/shared/testdata/unicode_v16.json'
        )
        tracked_pmcs = (
            name.startswith('.gen/miltech_ng/public/')
            and relative.name.startswith('user_pmcs_')
            and relative.suffix == '.go'
        )
        if not selected and not tracked_pmcs:
            continue
        source = validate_candidate_source(root, relative)
        digest = hashlib.sha256(source.read_bytes()).hexdigest()
        if selected:
            inputs[name] = digest
        if tracked_pmcs:
            pmcs[name] = digest
    return inputs, pmcs
