import importlib.util
from pathlib import Path
import unittest

spec=importlib.util.spec_from_file_location("gate",Path(__file__).with_name("enforce-runtime-vulnerabilities.py"))
gate=importlib.util.module_from_spec(spec);spec.loader.exec_module(gate)
REPO="PastureStack/node-agent"
class MetadataVex(unittest.TestCase):
    def setUp(self):
        self.finding={"VulnerabilityID":"GO-2026-5932","PkgName":"golang.org/x/crypto","InstalledVersion":"v0.58.0","Severity":"UNKNOWN","FixedVersion":"","PkgIdentifier":{"PURL":"pkg:golang/golang.org/x/crypto@v0.58.0"}}
        tools=('docker','dockerd','docker-proxy','containerd','containerd-shim-runc-v2','ctr','runc')
        self.report={"SchemaVersion":2,"Results":[{"Target":"usr/local/go/bin/go","Type":"gobinary","Packages":[{"Name":"stdlib"}]}]+[
            {"Target":"usr/bin/"+name,"Type":"gobinary","Packages":[{"Name":"stdlib"}],"Vulnerabilities":[self.finding] if name=='dockerd' else []} for name in tools]}
        self.ledger={"repository":REPO,"unfixed":[]}
        self.headers={"repository":REPO,"package":"linux-libc-dev","version":"7.0.0-38.38","purl":"pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04","ids":[]}
        graphs={name:'github.com/containerd/containerd/v2/cmd/'+name for name in ('containerd','ctr','containerd-shim-runc-v2')}
        graphs['runc']='github.com/opencontainers/runc'
        self.proof={"dockerdOpenpgp":{"binarySha256":"3a6132d4d99c1437e3422599f802207a3def9ba4de00ec1004a8e40d2740b9b0","sourceCommit":"0367cbeb49b9d98cb0b4f51e95f7b54b4803e76c","cryptoSymbols":779,"openpgpSymbols":0},"containerGraphs":graphs,
            "pip":{"verifiedWheelSha256":"be5f63d537bbe611dc422fabb6740a44b5997888b592888df5091727f023802a","pip":"26.2.2+pasturestack.1","urllib3":"2.8.0","vendorFilesByteCompared":299}}
    def check(self):
        return gate.builder_violations(self.report,self.ledger,REPO,{},self.headers,node_proof=self.proof)
    def test_exact_daemon_openpgp_absent(self): self.assertEqual(self.check(),[])
    def test_changed_binary_blocks(self):
        self.proof['dockerdOpenpgp']['binarySha256']='0'*64
        self.assertTrue(self.check())
    def test_missing_positive_control_blocks(self):
        self.proof['dockerdOpenpgp']['cryptoSymbols']=0
        self.assertTrue(self.check())
    def test_present_openpgp_blocks(self):
        self.proof['dockerdOpenpgp']['openpgpSymbols']=1
        self.assertTrue(self.check())
    def test_new_fixed_version_blocks(self):
        self.finding['FixedVersion']='0.59'
        self.assertTrue(self.check())
    def test_missing_actual_daemon_analysis_blocks(self):
        self.report['Results']=[r for r in self.report['Results'] if r['Target']!='usr/bin/dockerd']
        with self.assertRaises(ValueError): self.check()
    def test_missing_graph_root_blocks(self):
        self.proof['containerGraphs']['containerd']='fixture'
        with self.assertRaises(ValueError): self.check()
    def test_reachable_openpgp_graph_blocks(self):
        self.proof['containerGraphs']['runc']+='\ngolang.org/x/crypto/openpgp'
        with self.assertRaises(ValueError): self.check()
    def test_distribution_daemon_is_not_cli_exempt(self):
        self.finding.update(VulnerabilityID='CVE-2026-85747',PkgName='github.com/docker/distribution',InstalledVersion='v2.8.3+incompatible',Severity='MEDIUM',PkgIdentifier={'PURL':'pkg:golang/github.com/docker/distribution@v2.8.3%2Bincompatible'})
        self.assertTrue(self.check())
    def test_complete_patched_seed(self):
        self.finding.update(VulnerabilityID="CVE-2025-66418",PkgName="python3-pip-whl",InstalledVersion="25.1.1+dfsg-1ubuntu2",Severity='MEDIUM',PkgIdentifier={"PURL":"pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04"})
        self.assertEqual(self.check(),[])
        self.proof["pip"]["verifiedWheelSha256"]="0"*64
        self.assertTrue(self.check())

if __name__=="__main__": unittest.main()
