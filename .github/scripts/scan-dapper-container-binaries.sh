#!/usr/bin/env bash
# OS-package ownership must never suppress the executable dependency scan.
set -euo pipefail
test "$#" -eq 3
image=$1 scanner=$2 cache=$3
root=$(mktemp -d)
container=$(docker create "$image")
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf -- "$root"' EXIT
mkdir -p "$root/usr/bin" evidence/container-tools
for tool in docker dockerd docker-proxy containerd containerd-shim-runc-v2 ctr runc; do
    docker cp "$container:/usr/bin/$tool" "$root/usr/bin/$tool"
    test -s "$root/usr/bin/$tool"
done
docker cp "$container:/licenses/container-tools/." evidence/container-tools/
sha256sum "$root/usr/bin/dockerd" | awk '{print $1}' > evidence/dockerd.sha256
docker run --rm --entrypoint go "$image" tool nm /usr/bin/dockerd > evidence/dockerd-symbols.txt
python3 - <<'PY'
import hashlib, json
from pathlib import Path
lines=Path('evidence/dockerd-symbols.txt').read_text().splitlines()
crypto=[line for line in lines if 'golang.org/x/crypto/' in line]
openpgp=[line for line in crypto if 'golang.org/x/crypto/openpgp' in line]
sha=Path('evidence/dockerd.sha256').read_text().strip()
graph=Path('security/docker-daemon-package-graph.txt').read_bytes()
graph_sha=hashlib.sha256(graph).hexdigest()
assert graph_sha=='5b64dcb5265f1d7345e94688a0ee0700887000359cd25cc8f184d88835e79fd4'
packages=graph.decode().splitlines()
assert len(packages)==2078 and 'github.com/moby/moby/v2/cmd/dockerd' in packages
assert not any(p=='golang.org/x/crypto/openpgp' or p.startswith('golang.org/x/crypto/openpgp/') for p in packages)
assert sha=='3a6132d4d99c1437e3422599f802207a3def9ba4de00ec1004a8e40d2740b9b0'
assert len(crypto)>0 and not openpgp
Path('evidence/dockerd-openpgp-proof.json').write_text(json.dumps({
    'binarySha256':sha,'sourceCommit':'0367cbeb49b9d98cb0b4f51e95f7b54b4803e76c',
    'cryptoSymbols':len(crypto),'openpgpSymbols':len(openpgp),
    'sourcePackageGraphSha256':graph_sha}))
Path('evidence/container-tools/docker-daemon-package-graph.txt').write_bytes(graph)
PY
docker run --rm --network none -v "$root:/scan:ro" -v "$PWD/evidence:/evidence" \
    -v "$cache:/root/.cache/trivy" "$scanner" rootfs --skip-db-update --offline-scan \
    --scanners vuln,secret --list-all-pkgs --format json \
    --output /evidence/container-tools-security.json /scan
python3 - <<'PY'
import json
from pathlib import Path
report=json.loads(Path('evidence/container-tools-security.json').read_text())
results={r['Target']:r for r in report['Results'] if r.get('Type')=='gobinary'}
expected={'usr/bin/'+name for name in ('docker','dockerd','docker-proxy','containerd','containerd-shim-runc-v2','ctr','runc')}
assert set(results)==expected
assert all(results[target].get('Packages') for target in expected)
image=json.loads(Path('evidence/dapper-security-image.json').read_text())
image['Results']+=report['Results']
Path('evidence/dapper-security-raw.json').write_text(json.dumps(image))
PY
