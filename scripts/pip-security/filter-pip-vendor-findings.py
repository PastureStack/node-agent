#!/usr/bin/env python3
"""Apply narrowly proven pip-vendor VEX, retaining the original Trivy report."""
import argparse
from copy import deepcopy
from datetime import datetime, timezone
import json
import hashlib
import re
from pathlib import Path

WHEEL_SHA = "be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a"
VENDOR_INVENTORY_SHA = "144483e54419f59a5186652b932f7690d7b8bfaaab61ccc790c2881a89ab6212"
RULES = {
    "GHSA-6v7p-g79w-8964": ("msgpack", "1.1.2"),
    "CVE-2026-57585": ("msgpack", "1.1.2"),
    "CVE-2025-47273": ("setuptools", "70.3.0"),
    "CVE-2026-59890": ("setuptools", "70.3.0"),
    "GHSA-xmj3-9gf7-h52w": ("setuptools", "70.3.0"),
}


def namespace_wheel_absent(proof, sbom):
    rows = proof.get("vendorInventory", [])
    if proof.get("setuptoolsWheelModuleAbsent") is not True or len(rows) != 299:
        return False
    if not all(isinstance(row, dict) and set(row) == {"path", "sha256"}
               and isinstance(row["path"], str) and row["path"] and "\\" not in row["path"]
               and not row["path"].startswith("/") and ".." not in row["path"].split("/")
               and "setuptools" not in row["path"].split("/")
               and isinstance(row["sha256"], str) and re.fullmatch("[0-9a-f]{64}", row["sha256"])
               for row in rows):
        return False
    if len({row["path"] for row in rows}) != 299 or rows != sorted(rows, key=lambda row: row["path"]):
        return False
    digest = hashlib.sha256(json.dumps(rows, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
    if digest != proof.get("vendorInventorySha256") or digest != VENDOR_INVENTORY_SHA or sbom is None:
        return False
    document = json.loads(sbom)
    packages = [c for c in document.get("components", []) if c.get("purl") == "pkg:pypi/setuptools@70.3.0"]
    return document.get("bomFormat") == "CycloneDX" and packages and all(
        c.get("name") == "setuptools" and c.get("version") == "70.3.0" and c.get("type") == "library"
        for c in packages)


def apply(raw, proof, subject, sbom=None):
    valid = (proof["verifiedWheelSha256"] == WHEEL_SHA and
             proof["pip"] == "26.2.2+pasturestack.1" and proof["urllib3"] == "2.8.0" and
             proof["msgpackImplementation"] == "pip._vendor.msgpack.fallback" and
             proof["setuptoolsPackageAbsent"] is True and proof["vendorFilesByteCompared"] == 299)
    if not valid:
        raise ValueError("Exact installed pip vendor proof is required")
    reasons = {item["id"]: item for item in proof["notAffected"]}
    result = deepcopy(raw)
    statements = []
    for section in result.get("Results", []):
        retained = []
        for item in section.get("Vulnerabilities", []):
            key = item.get("VulnerabilityID")
            expected = RULES.get(key)
            actual = (item.get("PkgName"), item.get("InstalledVersion"))
            packages = [pkg for pkg in section.get("Packages", []) if
                        (pkg.get("Name"), pkg.get("Version")) == actual]
            reason = reasons.get(key, {})
            eligible = (expected == actual and key in reasons and
                        (reason.get("package"), reason.get("version")) == actual and
                        section.get("Type") == "python-pkg" and packages and
                        all(pkg.get("AnalyzedBy") == "sbom" for pkg in packages))
            if key == "GHSA-xmj3-9gf7-h52w":
                exact_purl = "pkg:pypi/setuptools@70.3.0"
                eligible = (eligible and item.get("PkgIdentifier", {}).get("PURL") == exact_purl
                            and all(pkg.get("Identifier", {}).get("PURL") == exact_purl for pkg in packages)
                            and namespace_wheel_absent(proof, sbom))
            if not eligible:
                retained.append(item)
                continue
            purl = "pkg:pypi/" + actual[0] + "@" + actual[1]
            statements.append({
                "vulnerability": {"name": key},
                "products": [{"@id": subject, "subcomponents": [{"@id": purl}]}],
                "status": "not_affected", "justification": "vulnerable_code_not_present",
                "impact_statement": reasons[key]["detail"] + "; exact pip wheel SHA-256 " + WHEEL_SHA,
            })
            if key == "GHSA-xmj3-9gf7-h52w":
                statements[-1]["impact_statement"] += ("; live/selected-wheel inventory SHA-256 " +
                    proof["vendorInventorySha256"] + "; native CycloneDX SHA-256 " + hashlib.sha256(sbom).hexdigest())
        section["Vulnerabilities"] = retained
    vex = {"@context": "https://openvex.dev/ns/v0.2.0", "@id": subject + "/pip-vendor-vex",
           "author": "PastureStack", "timestamp": datetime.now(timezone.utc).isoformat(),
           "version": 1, "statements": statements}
    return result, vex


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--raw", type=Path, required=True)
    parser.add_argument("--proof", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--vex", type=Path, required=True)
    parser.add_argument("--subject", required=True)
    parser.add_argument("--sbom", type=Path, help="Actual native CycloneDX bytes required for the namespace-wheel advisory")
    args = parser.parse_args()
    report, vex = apply(json.loads(args.raw.read_text()), json.loads(args.proof.read_text()), args.subject,
                        args.sbom.read_bytes() if args.sbom else None)
    args.output.write_text(json.dumps(report, indent=2) + "\n")
    args.vex.write_text(json.dumps(vex, indent=2) + "\n")
    print(f"PROVEN_PIP_VENDOR_VEX statements={len(vex['statements'])}")
