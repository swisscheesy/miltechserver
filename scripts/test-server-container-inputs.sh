#!/usr/bin/env bash
# Builds only a synthetic packaging fixture; never accepts a checkout as context.
set -euo pipefail

if [[ $# != 2 ]]; then
  echo 'Usage: test-server-container-inputs.sh <empty-temporary-workspace> <miltechserver-container-inputs-test:tag>' >&2
  exit 2
fi
workspace=$1
image=$2
if [[ ! $image =~ ^miltechserver-container-inputs-test:[a-zA-Z0-9_.-]+$ ]]; then
  echo 'Refusing image tag outside the disposable test namespace' >&2
  exit 2
fi
recipe_root=$(cd "$(dirname "$0")/.." && pwd -P)
# Check before touching the filesystem or contacting Docker.
python3 - "$workspace" <<'PY'
import os, pathlib, sys, tempfile
p = pathlib.Path(sys.argv[1]).absolute()
roots = [pathlib.Path('/tmp').resolve(), pathlib.Path('/private/tmp'), pathlib.Path(tempfile.gettempdir()).resolve()]
if p.is_symlink() or not any(root in p.resolve().parents for root in roots):
    sys.exit('Refusing workspace outside temporary storage or a symlink')
if p.exists() and (not p.is_dir() or any(p.iterdir())):
    sys.exit('Refusing nonempty workspace; existing checkout contexts are forbidden')
p.mkdir(parents=False, exist_ok=True)
PY
workspace=$(cd "$workspace" && pwd -P)

command -v docker >/dev/null || { echo 'UNRESOLVED: Docker CLI unavailable' >&2; exit 3; }
endpoint=${DOCKER_HOST:-}
if [[ -z $endpoint ]]; then
  endpoint=$(docker context inspect --format '{{.Endpoints.docker.Host}}')
fi
if [[ $endpoint != unix:///* ]]; then
  echo 'UNRESOLVED: refusing Docker endpoint other than a local Unix socket' >&2
  exit 3
fi
docker_local() {
  env -u DOCKER_CONTEXT -u DOCKER_HOST -u DOCKER_TLS_VERIFY -u DOCKER_CERT_PATH docker --host "$endpoint" "$@"
}
if ! python3 - "$endpoint" <<'PY'
import os, subprocess, sys
env = dict(os.environ)
for key in ('DOCKER_CONTEXT', 'DOCKER_HOST', 'DOCKER_TLS_VERIFY', 'DOCKER_CERT_PATH'):
    env.pop(key, None)
try:
    result = subprocess.run(['docker', '--host', sys.argv[1], 'info'], env=env,
                            stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, timeout=10)
    sys.exit(result.returncode)
except subprocess.TimeoutExpired:
    sys.exit(3)
PY
then
  echo 'UNRESOLVED: local Docker daemon unavailable (10-second bounded probe)' >&2
  exit 3
fi
for tag in "$image" "$image-backend" "$image-context"; do
  if docker_local image inspect "$tag" >/dev/null 2>&1; then
    echo 'Refusing to overwrite an existing test image' >&2
    exit 2
  fi
done

context="$workspace/context"
mkdir "$context"
cp "$recipe_root/Dockerfile" "$context/Dockerfile"
if [[ -f $recipe_root/.dockerignore ]]; then
  cp "$recipe_root/.dockerignore" "$context/.dockerignore"
fi
python3 - "$context" "$workspace" <<'PY'
import json, pathlib, sys
c, w = map(pathlib.Path, sys.argv[1:])
def write(path, contents):
    p = c / path
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(contents)
for path in ['.env', '.env.production', 'nested/.env', 'fire_auth_key.json',
             'nested/fire_auth_key.other.json', '.git/config', '.aws/credentials',
             '.codex/settings', '.agent/config', '.claude/config', '.gemini/config',
             '.superpowers/progress', '.codegraph/index', '.worktrees/other/file',
             '.gocache/cache', '__debug_bin',
             'server.log', 'logs/worker.log', 'nested/server.log', 'nested/logs/worker.log',
             'scratch/probe.txt', 'nested/scratch/probe.txt', 'tmp/probe.txt', 'nested/tmp/probe.txt',
             'temp/probe.txt', 'nested/temp/probe.txt', 'probe.tmp', 'nested/probe.tmp']:
    write(path, 'MILTECH_CONTAINER_SENTINEL_' + path)
write('go.mod', 'module containerfixture\n\ngo 1.23.0\n')
write('go.sum', '')
write('.gen/build_input.txt', 'GEN_BUILD_INPUT_PRESENT')
write('main.go', '''package main
import("encoding/json"; "fmt"; "os")
func main() {
  credentials, err := os.ReadFile(os.Getenv("FIREBASE_AUTH_KEY"))
  if err != nil || !json.Valid(credentials) { fmt.Println("fixture credentials unavailable"); os.Exit(1) }
  if err := os.WriteFile(".gen/runtime.txt", []byte("fixture generated"), 0600); err != nil { fmt.Println("fixture generation failed"); os.Exit(1) }
  fmt.Println("fixture startup generation calls: 1")
}
''')
write('frontend/package.json', json.dumps({'name':'containerfixture','version':'1.0.0','scripts':{'build':'node build.cjs'}}))
write('frontend/package-lock.json', json.dumps({'name':'containerfixture','version':'1.0.0','lockfileVersion':3,'packages':{'':{'name':'containerfixture','version':'1.0.0'}}}))
write('frontend/build.cjs', 'require("fs").mkdirSync("build"); require("fs").writeFileSync("build/index.html", "fixture");')
# This fake credential is outside the build context and is never an SDK key.
(w / 'runtime-credentials.json').write_text('{"fixture":true}')
(w / 'runtime-credentials.json').chmod(0o644)
write('Dockerfile.context', 'FROM scratch\nCOPY . /context/\n')
PY

containers=()
cleanup() {
  for container in "${containers[@]}"; do
    docker_local rm -f "$container" >/dev/null 2>&1 || echo 'WARNING: test container cleanup failed' >&2
  done
  docker_local image rm "$image" "$image-backend" "$image-context" >/dev/null 2>&1 \
    || echo 'WARNING: test image cleanup failed; some images may not have been built' >&2
}
trap cleanup EXIT

docker_local build --no-cache -f "$context/Dockerfile.context" -t "$image-context" "$context"
container=$(docker_local create "$image-context" /unused)
containers+=("$container")
docker_local export "$container" -o "$workspace/context.tar"
scan_archive() {
  python3 - "$1" <<'PY'
import io, sys, tarfile
matches = 0
def scan(stream):
    global matches
    with tarfile.open(fileobj=stream, mode='r:*') as archive:
        for member in archive:
            if not member.isfile(): continue
            data = archive.extractfile(member).read()
            if b'MILTECH_CONTAINER_SENTINEL_' in data: matches += 1
            try: scan(io.BytesIO(data))
            except tarfile.TarError: pass
with open(sys.argv[1], 'rb') as stream: scan(stream)
print(matches)
PY
}
sentinel_matches_in_context_and_layers=$(scan_archive "$workspace/context.tar")
if [[ $sentinel_matches_in_context_and_layers != 0 ]]; then
  echo "FAIL: $sentinel_matches_in_context_and_layers sentinel-bearing files entered the build context" >&2
  exit 1
fi
docker_local build --no-cache --target backend-builder -t "$image-backend" "$context"
docker_local build --no-cache -t "$image" "$context"
docker_local save "$image-backend" "$image" -o "$workspace/layers.tar"
sentinel_matches_in_context_and_layers=$(scan_archive "$workspace/layers.tar")
test "$sentinel_matches_in_context_and_layers" -eq 0
container=$(docker_local create "$image")
containers+=("$container")
docker_local export "$container" -o "$workspace/final.tar"
test "$(scan_archive "$workspace/final.tar")" -eq 0
python3 - "$workspace/final.tar" <<'PY'
import sys, tarfile
with tarfile.open(sys.argv[1]) as archive:
    assert archive.extractfile('app/.gen/build_input.txt').read() == b'GEN_BUILD_INPUT_PRESENT'
PY

mount="type=bind,source=$workspace/runtime-credentials.json,target=/run/secrets/firebase.json,readonly"
output=$(docker_local run --rm --network none --read-only --tmpfs /app/.gen:uid=10001,gid=10001,mode=0700 \
  --mount "$mount" -e FIREBASE_AUTH_KEY=/run/secrets/firebase.json "$image")
startup_generation_calls=$(printf '%s\n' "$output" | awk '/^fixture startup generation calls: 1$/ {count++} END {print count+0}')
test "$startup_generation_calls" -eq 1
set +e
docker_local run --rm --network none --read-only --tmpfs /app/.gen:uid=10001,gid=10001,mode=0700 \
  -e FIREBASE_AUTH_KEY=/run/secrets/firebase.json "$image" >"$workspace/missing-secret.log" 2>&1
missing_secret_startup_exit=$?
docker_local run --rm --network none --read-only --mount "$mount" \
  -e FIREBASE_AUTH_KEY=/run/secrets/firebase.json "$image" >"$workspace/unwritable-output.log" 2>&1
unwritable_output_startup_exit=$?
set -e
test "$missing_secret_startup_exit" -ne 0
test "$unwritable_output_startup_exit" -ne 0
grep -q '^fixture credentials unavailable$' "$workspace/missing-secret.log"
grep -q '^fixture generation failed$' "$workspace/unwritable-output.log"
echo "PASS: context/layer/final sentinels=0; fixture startup_generation_calls=1; missing secret and unwritable output fail"
echo 'Fixture packaging evidence only; real application SDK/Postgres startup remains a separate gate.'
