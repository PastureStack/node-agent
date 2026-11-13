import copy
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
        self.headers={"repository":REPO,"package":"linux-libc-dev","version":"7.0.0-43.43","purl":"pkg:deb/ubuntu/linux-libc-dev@7.0.0-43.43?arch=amd64&distro=ubuntu-26.04","ids":[]}
        graphs={name:'github.com/containerd/containerd/v2/cmd/'+name for name in ('containerd','ctr','containerd-shim-runc-v2')}
        graphs['runc']='github.com/opencontainers/runc'
        self.proof={"dockerdOpenpgp":{"binarySha256":"4e770837d688825898693db234f1723759b33650013af813eda20c21252ac7ad","sourceCommit":"70efc0e31da3a115fc87574d1b786fa2b867c346","sourceArchiveSha256":"0a58b26c1d14415c24c96b26423d53490d9e869aa26649fd820df8f74aaa6691","cryptoSymbols":779,"openpgpSymbols":0},"containerGraphs":graphs,
            "pip":{"verifiedWheelSha256":"2aef7c341eef310645ab56455f5789de2819463b25fa70151954e1d208d2db36","pip":"26.2.2+pasturestack.2","urllib3":"2.8.0","vendorFilesByteCompared":299}}
        self.proof['dockerdOpenpgp']['buildManifestSha256']=self.proof['dockerdOpenpgp']['binarySha256']
        self.proof['dockerdOpenpgp']['compiler']='go1.27.4'
        self.proof['dockerdOpenpgp']['sourcePackageGraphSha256']='9548a1b9b79f6dff64fe28ae4f0aac338af14bf84b71c3d1dbf9d1b47b11833f'
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

    def test_official_openpgp_fixed_marker_keeps_the_exact_absence_scope(self):
        self.finding.update(FixedVersion='0.58.1-0.20261009181718-2f1f834330b2', DataSource={'ID':'govulndb'})
        for target in ('usr/bin/dockerd','usr/bin/containerd','usr/bin/ctr','usr/bin/containerd-shim-runc-v2'):
            for row in self.report['Results']:
                row['Vulnerabilities']=[self.finding] if row['Target']==target else []
            with self.subTest(target=target): self.assertEqual(self.check(),[])
        self.proof['containerGraphs']['containerd']+='\ngithub.com/ProtonMail/go-crypto/openpgp'
        self.assertEqual(self.check(),[], 'The maintained implementation is not the affected Go package')

    def test_known_marker_still_rejects_changed_identity_or_absence_proof(self):
        self.finding.update(FixedVersion='0.58.1-0.20261009181718-2f1f834330b2', DataSource={'ID':'govulndb'})
        original_finding=copy.deepcopy(self.finding);original_proof=copy.deepcopy(self.proof)
        mutations=[{'VulnerabilityID':'GO-2026-OTHER'}, {'InstalledVersion':'v0.57.0'},
                   {'PkgIdentifier':{'PURL':'pkg:golang/golang.org/x/crypto@v0.57.0'}},
                   {'FixedVersion':'0.59.0'}, {'DataSource':{'ID':'other'}}]
        for mutation in mutations:
            with self.subTest(mutation=mutation):
                self.finding.clear();self.finding.update(original_finding,**mutation)
                self.assertTrue(self.check())
        self.finding.clear();self.finding.update(original_finding)
        for field,value in [('binarySha256','0'*64),('sourceCommit','0'*40),
                            ('sourcePackageGraphSha256','bad'),('compiler','go1.26.10'),('cryptoSymbols',0),('openpgpSymbols',1)]:
            self.proof=copy.deepcopy(original_proof);self.proof['dockerdOpenpgp'][field]=value
            with self.subTest(field=field):self.assertTrue(self.check())
        self.proof=copy.deepcopy(original_proof)
        self.report['Results'][1]['Vulnerabilities']=[self.finding]
        for row in self.report['Results'][2:]:row['Vulnerabilities']=[]
        self.assertTrue(self.check(), 'An unqualified target must remain rejected')
        for row in self.report['Results']:row['Vulnerabilities']=[self.finding] if row['Target']=='usr/bin/containerd' else []
        for graph in ('containerd','ctr','containerd-shim-runc-v2','runc'):
            self.proof=copy.deepcopy(original_proof);self.proof['containerGraphs'][graph]+='\ngolang.org/x/crypto/openpgp/packet'
            with self.subTest(graph=graph), self.assertRaises(ValueError):self.check()

    def test_fixed_pip_core_requires_the_exact_complete_new_byte_proof(self):
        for row in self.report['Results']:row['Vulnerabilities']=[]
        self.finding.update(VulnerabilityID='CVE-2026-8643',PkgName='python3-pip-whl',InstalledVersion='25.1.1+dfsg-1ubuntu2',
                            Severity='MEDIUM',PkgIdentifier={'PURL':'pkg:deb/ubuntu/python3-pip-whl@25.1.1%2Bdfsg-1ubuntu2?arch=all&distro=ubuntu-26.04'})
        os_result={'Target':'ubuntu','Class':'os-pkgs','Type':'ubuntu','Vulnerabilities':[self.finding]}
        self.report['Results'].append(os_result)
        self.assertTrue(self.check(), 'The historical vendor-only proof cannot establish patched pip core')
        self.proof['pip'].update(vendorInventorySha256='6b9e911363afa748740d5a514de1ecf34974645d0fb01dac23546e66d4d0f94c',
            pipCoreFilesByteCompared=157,pipCoreInventorySha256='51be05e205130ecb1655f762f5933177269a9e665d1f4075a8921557998ddce5',pipCoreFileSetMatched=True)
        self.assertEqual(self.check(),[])
        original=copy.deepcopy(self.proof['pip'])
        for field,value in [('verifiedWheelSha256','0'*64),('pip','26.2.2'),('vendorInventorySha256','0'*64),
                            ('pipCoreFilesByteCompared',156),('pipCoreInventorySha256','0'*64),('pipCoreFileSetMatched',False)]:
            self.proof['pip']=dict(original,**{field:value})
            with self.subTest(field=field):self.assertTrue(self.check())
        self.proof['pip']=original;before=copy.deepcopy(self.finding)
        for mutation in [{'VulnerabilityID':'CVE-2026-NEW'}, {'InstalledVersion':'25.1.1'},
                         {'PkgIdentifier':{'PURL':'pkg:pypi/pip@25.1.1'}}, {'FixedVersion':'26.1.3'}]:
            self.finding.clear();self.finding.update(before,**mutation)
            with self.subTest(mutation=mutation):self.assertTrue(self.check())
        self.finding.clear();self.finding.update(before);os_result['Type']='gobinary';os_result['Packages']=[{'Name':'stdlib'}]
        self.assertTrue(self.check())

if __name__=="__main__": unittest.main()
