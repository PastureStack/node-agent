import copy
import importlib.util
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location("gate",Path(__file__).with_name("enforce-runtime-vulnerabilities.py"))
gate=importlib.util.module_from_spec(spec);spec.loader.exec_module(gate)
REPO="PastureStack/node-agent"
class MetadataVex(unittest.TestCase):
    def setUp(self):
        self.finding={"VulnerabilityID":"CVE-2025-54388","PkgName":"docker.io","InstalledVersion":"29.1.3-0ubuntu4.1","Severity":"MEDIUM","FixedVersion":"","PkgIdentifier":{"PURL":"pkg:deb/ubuntu/docker.io@29.1.3-0ubuntu4.1?arch=amd64&distro=ubuntu-26.04"}}
        self.report={"SchemaVersion":2,"Results":[{"Target":"usr/local/go/bin/go","Type":"gobinary","Packages":[{"Name":"stdlib"}]},{"Target":"os","Vulnerabilities":[self.finding]}]}
        self.ledger={"repository":REPO,"unfixed":[]}
        self.headers={"repository":REPO,"package":"linux-libc-dev","version":"7.0.0-38.38","purl":"pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04","ids":[]}
        self.proof={"dockerdVersion":"Docker version 29.1.3, build fixture","pip":{"verifiedWheelSha256":"be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a","pip":"26.2.2+pasturestack.1","urllib3":"2.8.0","vendorFilesByteCompared":299}}
    def check(self):
        return gate.builder_violations(self.report,self.ledger,REPO,{},self.headers,node_proof=self.proof)
    def test_exact_daemon_outside_affected_range(self):
        self.assertEqual(self.check(),[])
    def test_changed_daemon_blocks(self):
        self.proof["dockerdVersion"]="Docker version 28.3.2, build fixture"
        self.assertTrue(self.check())
    def test_changed_package_blocks(self):
        self.finding["InstalledVersion"]="28.3.2"
        self.assertTrue(self.check())
    def test_available_fix_never_exempt(self):
        self.finding["FixedVersion"]="30"
        self.assertTrue(self.check())
    def test_complete_patched_seed(self):
        self.finding.update(VulnerabilityID="CVE-2025-66418",PkgName="python3-pip-whl",InstalledVersion="25.1.1+dfsg-1ubuntu2",PkgIdentifier={"PURL":"pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04"})
        self.assertEqual(self.check(),[])
        self.proof["pip"]["verifiedWheelSha256"]="0"*64
        self.assertTrue(self.check())

if __name__=="__main__": unittest.main()
