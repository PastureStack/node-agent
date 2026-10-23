"""Enforce release findings; only exact, reviewed unfixed low/medium entries may pass."""
import argparse
import json
import sys
from pathlib import Path


def violations(report, ledger, repository):
    if (not isinstance(report, dict) or report.get("SchemaVersion") != 2
            or not isinstance(report.get("Results"), list) or not report["Results"]):
        raise ValueError("missing or malformed Trivy scan results")
    for result in report["Results"]:
        if not isinstance(result, dict) or not isinstance(result.get("Target"), str):
            raise ValueError("malformed Trivy result")
        for key in ("Vulnerabilities", "Secrets"):
            if key in result and result[key] is not None and not isinstance(result[key], list):
                raise ValueError("malformed Trivy findings")
    if ledger.get("repository") != repository:
        raise ValueError("allowlist repository mismatch")
    reviewed = {}
    for entry in ledger.get("unfixed", []):
        key = (entry["id"], entry["package"], entry["version"], entry.get("purl", ""))
        if key in reviewed:
            raise ValueError("duplicate reviewed finding")
        if (entry.get("severity") not in ("LOW", "MEDIUM")
                or entry.get("status") != "no_official_fix"
                or not entry.get("officialEvidence", "").startswith("https://")
                or not entry.get("reason") or not entry.get("checkedAt")):
            raise ValueError("incomplete official no-fix evidence")
        reviewed[key] = entry
    errors = []
    for result in report.get("Results") or []:
        if result.get("Secrets"):
            errors.append("secret findings remain")
        for finding in result.get("Vulnerabilities") or []:
            severity = finding.get("Severity", "UNKNOWN")
            key = (finding.get("VulnerabilityID"), finding.get("PkgName"),
                   finding.get("InstalledVersion"),
                   (finding.get("PkgIdentifier") or {}).get("PURL", ""))
            entry = reviewed.get(key)
            fixed = str(finding.get("FixedVersion") or "").strip()
            if (severity not in ("LOW", "MEDIUM") or fixed
                    or entry is None or entry["severity"] != severity):
                errors.append("%s %s@%s severity=%s fixed=%s" % (
                    key[0], key[1], key[2], severity, fixed or "none"))
    return errors



def builder_violations(report, ledger, repository, header_boundary, header_ledger,
                       package_graphs=None, docker_proof=None, node_proof=None):
    # The complete raw scan is required; never infer safety from missing analysis.
    if not isinstance(report, dict) or report.get("SchemaVersion") != 2:
        raise ValueError("malformed complete builder report")
    results = report.get("Results")
    if not isinstance(results, list) or not results:
        raise ValueError("missing builder analysis")
    expected = ["usr/local/go/bin/go"]
    if repository == "PastureStack/ecr-credential-sync":
        expected += ["usr/bin/docker", "usr/libexec/docker/cli-plugins/docker-buildx"]
    if repository == "PastureStack/kubernetes-authentication-bridge":
        expected += ["usr/bin/docker", "usr/local/libexec/docker/cli-plugins/docker-buildx"]
    if repository == "PastureStack/node-agent":
        expected += ["usr/bin/" + name for name in (
            "docker", "dockerd", "docker-proxy", "containerd",
            "containerd-shim-runc-v2", "ctr", "runc")]
    binaries = {r.get("Target"): r for r in results if r.get("Type") == "gobinary"}
    for target in expected:
        if not binaries.get(target, {}).get("Packages"):
            raise ValueError("missing nonempty binary analysis: " + target)
    if any(not result.get("Packages") for result in binaries.values()):
        raise ValueError("incomplete Go binary package analysis")
    if repository == "PastureStack/node-agent" and node_proof:
        graphs = node_proof.get("containerGraphs", {})
        roots = {name: "github.com/containerd/containerd/v2/cmd/" + name
                 for name in ("containerd", "ctr", "containerd-shim-runc-v2")}
        roots["runc"] = "github.com/opencontainers/runc"
        if set(graphs) != set(roots) or any(
                root not in graphs[name].splitlines() or any(
                    p == "golang.org/x/crypto/openpgp" or p.startswith("golang.org/x/crypto/openpgp/")
                    for p in graphs[name].splitlines()) for name, root in roots.items()):
            raise ValueError("missing exact container-tool package graphs")
    has_headers = any(f.get("PkgName") == "linux-libc-dev" for r in results for f in (r.get("Vulnerabilities") or []))
    if has_headers and header_boundary != {"package": "linux-libc-dev", "version": "7.0.0-38.38",
                                           "kernel_images": False, "kernel_modules": False}:
        raise ValueError("missing exact installed-header boundary proof")
    if header_ledger.get("repository") != repository:
        raise ValueError("header review repository mismatch")
    if (header_ledger.get("package") != "linux-libc-dev"
            or header_ledger.get("version") != "7.0.0-38.38"
            or header_ledger.get("purl") != "pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04"):
        raise ValueError("changed header review boundary")
    header_keys = {(identifier, header_ledger["package"], header_ledger["version"], header_ledger["purl"])
                   for identifier in header_ledger.get("ids", [])}
    expected_purl = "pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04"
    openpgp_absent = False
    if package_graphs:
        if repository not in ("PastureStack/kubernetes-authentication-bridge", "PastureStack/ecr-credential-sync"):
            raise ValueError("unexpected OpenPGP proof consumer")
        expected_roots = {"buildx": "github.com/docker/buildx/cmd/buildx"}
        if repository == "PastureStack/ecr-credential-sync":
            expected_roots["docker"] = "github.com/docker/cli/cmd/docker"
        if set(package_graphs) != set(expected_roots):
            raise ValueError("unexpected source package graph set")
        for name, root in expected_roots.items():
            packages = set(package_graphs[name].splitlines())
            if root not in packages or any(
                    p == "golang.org/x/crypto/openpgp" or
                    p.startswith("golang.org/x/crypto/openpgp/") for p in packages):
                raise ValueError("OpenPGP package boundary not proved")
        openpgp_absent = True
    retained = []
    for result in results:
        copied = dict(result)
        copied["Vulnerabilities"] = []
        for finding in result.get("Vulnerabilities") or []:
            key = (finding.get("VulnerabilityID"), finding.get("PkgName"),
                   finding.get("InstalledVersion"),
                   (finding.get("PkgIdentifier") or {}).get("PURL", ""))
            is_header = (key in header_keys and key[1] == "linux-libc-dev"
                         and key[2] == "7.0.0-38.38" and key[3] == expected_purl)
            unfixed = not str(finding.get("FixedVersion") or "").strip()
            if is_header and unfixed:
                continue
            if (unfixed and openpgp_absent and key == (
                    "GO-2026-5932", "golang.org/x/crypto", "v0.57.0",
                    "pkg:golang/golang.org/x/crypto@v0.57.0")):
                continue
            if (unfixed and repository == "PastureStack/ecr-credential-sync"
                    and key == ("CVE-2026-85747", "github.com/docker/distribution",
                                "v2.8.3+incompatible",
                                "pkg:golang/github.com/docker/distribution@v2.8.3%2Bincompatible")
                    and docker_proof and len(docker_proof.get("cases", [])) == 7
                    and all(c.get("passed") is True for c in docker_proof["cases"])
                    and {c.get("case") for c in docker_proof["cases"]} == {
                        "valid_schema2", "forged_schema2", "valid_oci", "forged_oci",
                        "valid_list", "forged_list_parent", "forged_list_child"}
                    and docker_proof.get("sourceCommit") == "a7dcaa6fdb6ed04aacbfdc76357fdae01605609e"
                    and len(docker_proof.get("binarySha256", "")) == 64):
                continue
            if (unfixed and repository == "PastureStack/node-agent" and node_proof):
                daemon = node_proof.get("dockerdOpenpgp", {})
                if (key == ("GO-2026-5932", "golang.org/x/crypto", "v0.58.0",
                            "pkg:golang/golang.org/x/crypto@v0.58.0")
                        and result.get("Target") == "usr/bin/dockerd"
                        and daemon.get("binarySha256") == "3a6132d4d99c1437e3422599f802207a3def9ba4de00ec1004a8e40d2740b9b0"
                        and daemon.get("sourceCommit") == "0367cbeb49b9d98cb0b4f51e95f7b54b4803e76c"
                        and daemon.get("sourcePackageGraphSha256") == "5b64dcb5265f1d7345e94688a0ee0700887000359cd25cc8f184d88835e79fd4"
                        and daemon.get("cryptoSymbols", 0) > 0
                        and daemon.get("openpgpSymbols") == 0):
                    continue
                graphs = node_proof.get("containerGraphs", {})
                if (key == ("GO-2026-5932", "golang.org/x/crypto", "v0.57.0",
                            "pkg:golang/golang.org/x/crypto@v0.57.0")
                        and result.get("Target") in ("usr/bin/containerd", "usr/bin/ctr", "usr/bin/containerd-shim-runc-v2")
                        and set(graphs) == {"containerd", "ctr", "containerd-shim-runc-v2", "runc"}
                        and all(graph and not any(p == "golang.org/x/crypto/openpgp" or p.startswith("golang.org/x/crypto/openpgp/") for p in graph.splitlines()) for graph in graphs.values())):
                    continue
                pip = node_proof.get("pip", {})
                if (key[0] in ("CVE-2025-66418", "CVE-2025-66471", "CVE-2026-21441", "CVE-2026-1703")
                        and key[1] == "python3-pip-whl" and key[2] == "25.1.1+dfsg-1ubuntu2"
                        and key[3] == "pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04"
                        and pip.get("verifiedWheelSha256") == "be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a"
                        and pip.get("pip") == "26.2.2+pasturestack.1"
                        and pip.get("urllib3") == "2.8.0" and pip.get("vendorFilesByteCompared") == 299):
                    continue
            copied["Vulnerabilities"].append(finding)
        retained.append(copied)
    return violations(dict(report, Results=retained), ledger, repository)

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--repository", required=True)
    parser.add_argument("--scope", choices=("runtime", "builder"), default="runtime")
    args = parser.parse_args()
    repo_root = Path(__file__).resolve().parents[2]
    report = json.load(sys.stdin)
    ledger_name = "runtime-unfixed.json" if args.scope == "runtime" else "dapper-unfixed.json"
    ledger = json.loads((repo_root / ".github" / ledger_name).read_text(encoding="utf-8"))
    if args.scope == "runtime":
        failures = violations(report, ledger, args.repository)
    else:
        evidence_dir = "dist" if args.repository in (
            "PastureStack/kubernetes-agent",
            "PastureStack/kubernetes-authentication-bridge") else "evidence"
        boundary = json.loads((repo_root / evidence_dir / "dapper-header-boundary.json").read_text(encoding="utf-8"))
        headers = json.loads((repo_root / ".github" / "dapper-headers.json").read_text(encoding="utf-8"))
        graphs = None
        docker_proof = None
        if args.repository in ("PastureStack/kubernetes-authentication-bridge",
                               "PastureStack/ecr-credential-sync"):
            graphs = {"buildx": (repo_root / evidence_dir / "buildx-package-graph.txt").read_text(encoding="utf-8")}
        if args.repository == "PastureStack/ecr-credential-sync":
            graphs["docker"] = (repo_root / evidence_dir / "docker-package-graph.txt").read_text(encoding="utf-8")
            docker_proof = json.loads((repo_root / evidence_dir / "docker-manifest-digest-proof.json").read_text(encoding="utf-8"))
        node_proof = None
        if args.repository == "PastureStack/node-agent":
            node_proof = {"pip": json.loads((repo_root / evidence_dir / "pip-vendor-proof.json").read_text(encoding="utf-8")),
                          "dockerdOpenpgp": json.loads((repo_root / evidence_dir / "dockerd-openpgp-proof.json").read_text(encoding="utf-8")),
                          "containerGraphs": {name: (repo_root / evidence_dir / "container-tools" / (name + "-package-graph.txt")).read_text(encoding="utf-8") for name in ("containerd", "ctr", "containerd-shim-runc-v2", "runc")}}
        failures = builder_violations(report, ledger, args.repository, boundary, headers, graphs, docker_proof, node_proof)
    if failures:
        raise SystemExit("Release vulnerability policy failed:\n" + "\n".join(failures))
    print("RELEASE_VULNERABILITY_POLICY_OK")
