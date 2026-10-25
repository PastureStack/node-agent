#!/usr/bin/env python3
"""Negative controls for the exact pip-vendor VEX boundary."""
from copy import deepcopy
import importlib.util
from pathlib import Path
import hashlib
import json
import tempfile
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location("pip_vex", Path(__file__).with_name("filter-pip-vendor-findings.py"))
vex = importlib.util.module_from_spec(spec)
spec.loader.exec_module(vex)
proof_spec = importlib.util.spec_from_file_location("pip_vendor_proof", Path(__file__).with_name("check-pip-vendor-slice.py"))
vendor_proof = importlib.util.module_from_spec(proof_spec)
proof_spec.loader.exec_module(vendor_proof)


class VexBoundary(unittest.TestCase):
    def setUp(self):
        self.proof = {"verifiedWheelSha256": vex.WHEEL_SHA, "pip": "26.2.2+pasturestack.1",
                      "urllib3": "2.8.0", "msgpackImplementation": "pip._vendor.msgpack.fallback",
                      "setuptoolsPackageAbsent": True, "vendorFilesByteCompared": 299,
                      "notAffected": [{"id": key, "package": value[0], "version": value[1],
                                       "detail": "Test fixture only"} for key, value in vex.RULES.items()]}
        self.proof["setuptoolsWheelModuleAbsent"] = True
        self.proof["vendorInventory"] = [{"path": f"fixture/{n:03d}.py", "sha256": "a" * 64} for n in range(299)]
        self.rehash_inventory(self.proof)
        # This synthetic metadata fixture is not a claim about installed files.
        anchor = patch.object(vex, "VENDOR_INVENTORY_SHA", self.proof["vendorInventorySha256"])
        anchor.start(); self.addCleanup(anchor.stop)
        self.sbom = json.dumps({"bomFormat": "CycloneDX", "components": [{"type": "library", "name": "setuptools",
            "version": "70.3.0", "purl": "pkg:pypi/setuptools@70.3.0"}]}).encode()
        self.raw = {"Results": [{"Type": "python-pkg", "Packages": [
            {"Name": name, "Version": version, "AnalyzedBy": "sbom", "Identifier": {"PURL": "pkg:pypi/" + name + "@" + version}}
            for name, version in set(vex.RULES.values())], "Vulnerabilities": [
            {"VulnerabilityID": key, "PkgName": value[0], "InstalledVersion": value[1],
             "PkgIdentifier": {"PURL": "pkg:pypi/" + value[0] + "@" + value[1]}}
            for key, value in vex.RULES.items()]}]}

    def test_exact_slice_only(self):
        original = deepcopy(self.raw)
        result, document = vex.apply(self.raw, self.proof, "urn:test:exact-image", self.sbom)
        self.assertEqual(result["Results"][0]["Vulnerabilities"], [])
        self.assertEqual(len(document["statements"]), 5)
        self.assertEqual(self.raw, original, "raw evidence must not be mutated")

    def test_invalid_proof_fails_closed(self):
        for key, value in [("verifiedWheelSha256", "0" * 64), ("urllib3", "2.7.0"),
                           ("vendorFilesByteCompared", 300), ("setuptoolsPackageAbsent", False)]:
            proof = dict(self.proof, **{key: value})
            with self.subTest(key=key), self.assertRaises(ValueError):
                vex.apply(self.raw, proof, "urn:test:image", self.sbom)

    def test_full_distribution_is_not_excluded(self):
        for package in self.raw["Results"][0]["Packages"]:
            package["AnalyzedBy"] = "python-pkg"
        result, document = vex.apply(self.raw, self.proof, "urn:test:image", self.sbom)
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), 5)
        self.assertEqual(document["statements"], [])

    def test_missing_packages_is_not_excluded(self):
        self.raw["Results"][0]["Packages"] = []
        result, _ = vex.apply(self.raw, self.proof, "urn:test:image", self.sbom)
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), 5)

    def test_wrong_version_is_not_excluded(self):
        for finding in self.raw["Results"][0]["Vulnerabilities"]:
            finding["InstalledVersion"] = "999.0"
        result, document = vex.apply(self.raw, self.proof, "urn:test:image", self.sbom)
        self.assertEqual(len(result["Results"][0]["Vulnerabilities"]), 5)
        self.assertEqual(document["statements"], [])

    def test_unknown_advisory_and_urllib3_are_retained(self):
        unknown = {"VulnerabilityID": "CVE-TEST-UNKNOWN", "PkgName": "setuptools", "InstalledVersion": "70.3.0"}
        urllib = {"VulnerabilityID": "CVE-2026-97689", "PkgName": "urllib3", "InstalledVersion": "2.7.0"}
        self.raw["Results"][0]["Vulnerabilities"] += [unknown, urllib]
        result, _ = vex.apply(self.raw, self.proof, "urn:test:image", self.sbom)
        self.assertEqual(result["Results"][0]["Vulnerabilities"], [unknown, urllib])

    @staticmethod
    def rehash_inventory(proof):
        proof["vendorInventorySha256"] = hashlib.sha256(json.dumps(
            proof["vendorInventory"], sort_keys=True, separators=(",", ":")).encode()).hexdigest()

    def test_actual_wheel_module_presence_fails_source_proof(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            vendor_proof.require_setuptools_wheel_absent([], root)
            with self.assertRaisesRegex(AssertionError, "selected wheel"):
                vendor_proof.require_setuptools_wheel_absent(["pip/_vendor/setuptools/wheel.py"], root)
            (root / "setuptools").mkdir(); (root / "setuptools/wheel.py").write_text("class Wheel: pass\n")
            with self.assertRaisesRegex(AssertionError, "installed"):
                vendor_proof.require_setuptools_wheel_absent([], root)

    def test_new_advisory_requires_inventory_module_absence_and_native_sbom(self):
        for kind in ("missing-proof", "module-present", "inventory-tamper", "missing-sbom", "wrong-sbom", "finding-purl", "package-purl"):
            proof = deepcopy(self.proof); sbom = self.sbom; raw = deepcopy(self.raw)
            if kind == "missing-proof": proof.pop("setuptoolsWheelModuleAbsent")
            if kind == "module-present":
                proof["vendorInventory"][-1]["path"] = "setuptools/wheel.py"
                proof["vendorInventory"].sort(key=lambda row: row["path"]); self.rehash_inventory(proof)
            if kind == "inventory-tamper": proof["vendorInventory"][0]["sha256"] = "b" * 64
            if kind == "missing-sbom": sbom = None
            if kind == "wrong-sbom": sbom = b'{"bomFormat":"CycloneDX","components":[]}'
            if kind == "finding-purl":
                next(v for v in raw["Results"][0]["Vulnerabilities"] if v["VulnerabilityID"] == "GHSA-xmj3-9gf7-h52w")["PkgIdentifier"]["PURL"] = "pkg:pypi/setuptools@999"
            if kind == "package-purl":
                next(p for p in raw["Results"][0]["Packages"] if p["Name"] == "setuptools")["Identifier"]["PURL"] = "pkg:pypi/setuptools@999"
            result, document = vex.apply(raw, proof, "urn:test:image", sbom)
            with self.subTest(kind=kind):
                self.assertEqual([v["VulnerabilityID"] for v in result["Results"][0]["Vulnerabilities"]], ["GHSA-xmj3-9gf7-h52w"])
                self.assertEqual(len(document["statements"]), 4)


if __name__ == "__main__":
    unittest.main()
