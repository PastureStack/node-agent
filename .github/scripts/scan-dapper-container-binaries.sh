#!/usr/bin/env bash
# OS-package ownership must never suppress the executable dependency scan.
set -euo pipefail
test "$#" -eq 3
image=$1 scanner=$2 cache=$3
root=$(mktemp -d)
container=$(docker create "$image")
trap 'docker rm -f "$container" >/dev/null 2>&1 || true; rm -rf -- "$root"' EXIT
mkdir -p "$root/usr/bin" "$root/usr/local/lib/docker/cli-plugins" evidence/container-tools
for tool in docker dockerd docker-proxy containerd containerd-shim-runc-v2 ctr runc; do
    docker cp "$container:/usr/bin/$tool" "$root/usr/bin/$tool"
    test -s "$root/usr/bin/$tool"
done
docker cp "$container:/licenses/container-tools/." evidence/container-tools/
docker cp "$container:/usr/local/lib/docker/cli-plugins/docker-buildx" "$root/usr/local/lib/docker/cli-plugins/docker-buildx"
test -s "$root/usr/local/lib/docker/cli-plugins/docker-buildx"
docker run --rm --entrypoint sh "$image" -lc 'test ! -e /usr/libexec/docker/cli-plugins/docker-buildx'
for suffix in SOURCE.tsv PACKAGES.txt SOURCE-FILES.tsv MODULES-before.txt MODULES-after.txt SOURCE-GOVULNCHECK.json GO-2026-5617.json GO-2026-5746.json GO-2026-6253.json GO-2026-6611.json; do
    docker cp "$container:/licenses/BUILDX-$suffix" "evidence/BUILDX-$suffix"
done
docker run --rm --entrypoint go "$image" version -m /usr/local/lib/docker/cli-plugins/docker-buildx > evidence/buildx-go-version.txt
docker run --rm --entrypoint docker "$image" buildx version > evidence/buildx-version.txt

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
expected={'usr/bin/'+name for name in ('docker','dockerd','docker-proxy','containerd','containerd-shim-runc-v2','ctr','runc')} | {'usr/local/lib/docker/cli-plugins/docker-buildx'}
assert set(results)==expected
assert all(results[target].get('Packages') for target in expected)
image=json.loads(Path('evidence/dapper-security-image.json').read_text())
image['Results']+=report['Results']
Path('evidence/dapper-security-raw.json').write_text(json.dumps(image))
PY

# Qualify only this newly built tool; the original complete raw reports remain intact.
buildx_sha="$(sha256sum "$root/usr/local/lib/docker/cli-plugins/docker-buildx")"
buildx_sha="${buildx_sha%% *}"
jq -n \
  --rawfile binding evidence/BUILDX-SOURCE.tsv \
  --rawfile packages evidence/BUILDX-PACKAGES.txt \
  --rawfile files evidence/BUILDX-SOURCE-FILES.tsv \
  --rawfile modulesBefore evidence/BUILDX-MODULES-before.txt \
  --rawfile modulesAfter evidence/BUILDX-MODULES-after.txt \
  --rawfile buildInfo evidence/buildx-go-version.txt \
  --rawfile version evidence/buildx-version.txt \
  --rawfile sourceCheck evidence/BUILDX-SOURCE-GOVULNCHECK.json \
  --rawfile daemon42306 evidence/BUILDX-GO-2026-5617.json \
  --rawfile daemon41567 evidence/BUILDX-GO-2026-5746.json \
  --rawfile archive17106 evidence/BUILDX-GO-2026-6253.json \
  --rawfile net78669 evidence/BUILDX-GO-2026-6611.json \
  --rawfile raw evidence/dapper-security-raw.json \
  --slurpfile vex security/buildx.openvex.json \
  --arg binarySha256 "$buildx_sha" \
  '{binding:$binding,packages:$packages,files:$files,modulesBefore:$modulesBefore,modulesAfter:$modulesAfter,buildInfo:$buildInfo,version:$version,sourceCheck:$sourceCheck,osvInputs:{"GO-2026-5617":$daemon42306,"GO-2026-5746":$daemon41567,"GO-2026-6253":$archive17106,"GO-2026-6611":$net78669},raw:$raw,vex:$vex[0],binarySha256:$binarySha256}' \
  | python3 -B .github/scripts/check_buildx_code_absence.py > evidence/buildx-code-absence.json

# Match the raw image and standalone-ELF scanner capabilities exactly; only VEX/output differ.
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
    -v "$PWD:/work" -w /work -v "$cache:/root/.cache/trivy" "$scanner" image \
    --skip-db-update --offline-scan --scanners vuln,secret --list-all-pkgs \
    --vex /work/security/buildx.openvex.json --format json \
    --output /work/evidence/dapper-security-buildx-image.json "$image"
docker run --rm --network none -v "$root:/scan:ro" -v "$PWD/evidence:/evidence" \
    -v "$PWD/security:/vex:ro" -v "$cache:/root/.cache/trivy" "$scanner" rootfs \
    --skip-db-update --offline-scan --scanners vuln,secret --list-all-pkgs \
    --vex /vex/buildx.openvex.json --format json \
    --output /evidence/container-tools-security-buildx.json /scan
python3 - <<'PY'
import json
from pathlib import Path
image=json.loads(Path('evidence/dapper-security-buildx-image.json').read_text())
tools=json.loads(Path('evidence/container-tools-security-buildx.json').read_text())
image['Results']+=tools['Results']
Path('evidence/dapper-security-buildx.json').write_text(json.dumps(image))
PY
jq -n --rawfile raw evidence/dapper-security-raw.json \
    --rawfile effective evidence/dapper-security-buildx.json \
    --slurpfile guard evidence/buildx-code-absence.json \
    --slurpfile vex security/buildx.openvex.json \
    '{raw:$raw,effective:$effective,guard:$guard[0],vex:$vex[0]}' \
    | python3 -B .github/scripts/check_buildx_code_absence.py --native > evidence/buildx-native-vex.json
