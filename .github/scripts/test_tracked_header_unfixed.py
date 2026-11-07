"""Source-unit fixtures only; these are not an actual header-43 scan or release PASS."""
import copy
from datetime import datetime, timedelta, timezone
import hashlib
import importlib.util
import json
from pathlib import Path
import unittest

HERE = Path(__file__).resolve().parent
spec = importlib.util.spec_from_file_location("policy", HERE / "enforce-runtime-vulnerabilities.py")
policy = importlib.util.module_from_spec(spec)
spec.loader.exec_module(policy)
REPO = "PastureStack/node-agent"


class TrackedHeaderPolicy(unittest.TestCase):
    def setUp(self):
        self.registry = json.loads((HERE.parent / "dapper-header-tracked-unfixed.json").read_text(encoding="utf-8"))
        today = datetime.now(timezone.utc).date()
        self.registry.update(status="no_official_fix", checkedAt=today.isoformat(),
                             nextReview=(today + timedelta(days=7)).isoformat(),
                             ids=["CVE-2026-98890"],
                             idsSha256=hashlib.sha256(b"CVE-2026-98890\n").hexdigest(),
                             scanEvidence={"status": "ORIGINAL_HEADER_ONLY_SCAN_RECORDED", "originalRunId": 1,
                                           "artifactId": 2, "sourceRevision": "a" * 40,
                                           "sha256": "b" * 64, "productQualification": False})
        source = self.registry["packageSource"]
        self.headers = {"repository": REPO, "package": source["package"], "version": source["version"],
                        "purl": source["purl"], "ids": ["CVE-2026-0001"]}
        self.recipe = "ADD --checksum=sha256:%s %s /tmp/linux-libc-dev.deb\n" % (source["debSha256"], source["debUrl"])
        self.ledger = {"repository": REPO, "unfixed": []}
        self.finding = {"VulnerabilityID": "CVE-2026-98890", "PkgName": source["package"],
                        "InstalledVersion": source["version"], "PkgIdentifier": {"PURL": source["purl"]},
                        "Severity": "MEDIUM", "FixedVersion": ""}
        self.report = {"SchemaVersion": 2, "Results": [{"Target": "ubuntu", "Vulnerabilities": [self.finding]}]}

    def merge(self):
        return policy.merge_tracked_header_unfixed(self.ledger, self.registry, self.headers, self.recipe)

    def test_exact_medium_uses_normal_tracking_without_changing_raw_or_code_absent_ledger(self):
        before = copy.deepcopy((self.ledger, self.headers, self.registry, self.report))
        merged = self.merge()
        self.assertEqual(policy.violations(self.report, merged, REPO), [])
        self.assertEqual(merged["unfixed"][0]["trackingStatus"], "tracked_unfixed")
        self.assertEqual(merged["unfixed"][0]["debSha256"], self.registry["packageSource"]["debSha256"])
        self.assertNotIn("justification", merged["unfixed"][0])
        self.assertEqual((self.ledger, self.headers, self.registry, self.report), before)

    def test_high_critical_fixable_unknown_or_other_identity_still_rejected(self):
        merged = self.merge()
        variants = [{"Severity": severity} for severity in ("HIGH", "CRITICAL", "LOW", "UNKNOWN")]
        variants += [{"FixedVersion": fixed} for fixed in ("7.0.0-44.44", " ", 1)]
        variants += [{"VulnerabilityID": "CVE-2026-999999"}, {"PkgName": "linux-image"},
                     {"InstalledVersion": "7.0.0-42.42"}, {"PkgIdentifier": {"PURL": "pkg:other/header@1"}}]
        for mutation in variants:
            with self.subTest(mutation=mutation):
                self.report["Results"][0]["Vulnerabilities"] = [dict(self.finding, **mutation)]
                self.assertTrue(policy.violations(self.report, merged, REPO))
        self.report["Results"][0]["Vulnerabilities"] = [self.finding]
        self.report["Results"][0]["Secrets"] = [{"RuleID": "secret"}]
        self.assertTrue(policy.violations(self.report, merged, REPO))

    def test_registration_source_status_review_or_cohort_drift_is_rejected(self):
        original = copy.deepcopy(self.registry)
        mutations = [{"status": "PENDING_ACTUAL_HEADER43_SCAN"}, {"classification": "code_absent"},
                     {"justification": "vulnerable_code_not_present"},
                     {"severity": "HIGH"}, {"fixedVersion": "7.0.0-44.44"},
                     {"ids": ["CVE-2026-999999"]}, {"ids": self.registry["ids"] * 2},
                     {"idsSha256": "0" * 64}, {"repository": "PastureStack/other"},
                     {"nextReview": (datetime.now(timezone.utc).date() - timedelta(days=1)).isoformat()},
                     {"scanEvidence": dict(self.registry["scanEvidence"], productQualification=True)},
                     {"scanEvidence": dict(self.registry["scanEvidence"], sha256=None)}]
        for field in ("version", "purl", "debSha256", "debUrl", "signedIndexSha256", "inReleaseSha256"):
            mutations.append({"packageSource": dict(self.registry["packageSource"], **{field: "wrong"})})
        for mutation in mutations:
            with self.subTest(mutation=mutation):
                self.registry = dict(copy.deepcopy(original), **mutation)
                with self.assertRaises(ValueError): self.merge()
        self.registry = original
        for recipe in ("", self.recipe.replace("3a3adc", "0a3adc"), self.recipe * 2):
            self.recipe = recipe
            with self.assertRaises(ValueError): self.merge()
        self.recipe = "ADD --checksum=sha256:%s %s /tmp/linux-libc-dev.deb\n" % (
            original["packageSource"]["debSha256"], original["packageSource"]["debUrl"])
        self.headers["ids"] = list(self.registry["ids"])
        with self.assertRaises(ValueError): self.merge()

    def test_unmaterialized_candidate_fails_closed(self):
        pending = json.loads((HERE.parent / "dapper-header-tracked-unfixed.json").read_text(encoding="utf-8"))
        pending["status"] = "PENDING_ACTUAL_HEADER43_SCAN"
        with self.assertRaises(ValueError):
            policy.merge_tracked_header_unfixed(self.ledger, pending, self.headers, self.recipe)


if __name__ == "__main__":
    unittest.main()
