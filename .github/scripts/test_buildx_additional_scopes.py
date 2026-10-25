#!/usr/bin/env python3
"""Per-ID additional source scopes cannot exempt another target or record."""
import copy
import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch

import check_buildx_additional_scopes as additional
import check_buildx_code_absence as shared
from test_buildx_code_absence import fixture

spec = importlib.util.spec_from_file_location('policy', Path(__file__).with_name('enforce-runtime-vulnerabilities.py'))
policy = importlib.util.module_from_spec(spec); spec.loader.exec_module(policy)


def data_fixture():
    source = fixture()
    for field in ['modulesBefore', 'modulesAfter']:
        source[field] += 'golang.org/x/crypto v0.57.0\n'
    source['buildInfo'] += '\tdep\tgolang.org/x/crypto\tv0.57.0\th1:fixture\n'
    checks = list(shared.objects(source['sourceCheck']))
    next(x['SBOM'] for x in checks if 'SBOM' in x)['modules'].append(
        {'path': 'golang.org/x/crypto', 'version': 'v0.57.0'})
    source['sourceCheck'] = '\n'.join(json.dumps(x) for x in checks)
    raw = json.loads(source['raw'])
    for identifier, (package, version, purl) in additional.SCOPES.items():
        raw['Results'][0]['Vulnerabilities'].append({'VulnerabilityID': identifier, 'PkgName': package,
            'InstalledVersion': version, 'PkgIdentifier': {'PURL': purl, 'UID': 'original'},
            'Severity': 'UNKNOWN' if identifier.startswith('GO-') else 'MEDIUM',
            'FixedVersion': '29.3.1' if identifier == 'CVE-2026-33997' else ''})
    source['raw'] = json.dumps(raw)
    root = Path(__file__).resolve().parents[2] / 'security/buildx-source-scopes'
    return {'source': source, 'baseGuard': shared.source_check(source),
            'osv': {identifier: (root / (identifier + '.json')).read_text(encoding='utf-8') for identifier in additional.INPUT_SHAS},
            'pluginFix': (root / 'CVE-2026-33997-official-commit.json').read_text(encoding='utf-8')}


class AdditionalScopes(unittest.TestCase):
    def setUp(self):
        hashes = {k: shared.sha(v) for k, v in fixture()['osvInputs'].items()}
        anchor = patch.dict(shared.OSV_PINS, hashes, clear=True)
        anchor.start(); self.addCleanup(anchor.stop)

    def test_three_exact_scopes_preserve_native_four_id_claim(self):
        data = data_fixture(); result = additional.qualify(data)
        self.assertEqual({r['id'] for r in result['notAffected']}, set(additional.SCOPES))
        self.assertEqual(result['sourceGuard']['canonicalOsvSha256'], shared.OSV_PINS)
        self.assertTrue(all(r['status'] == 'not_affected' for r in result['notAffected']))

    def test_missing_or_tampered_canonical_and_official_patch_rejected(self):
        for field in ['missing', 'canonical', 'patch', 'base']:
            data = data_fixture()
            if field == 'missing': data['osv'].pop('GO-2026-4883')
            if field == 'canonical': data['osv']['GO-2026-5932'] += '\n'
            if field == 'patch': data['pluginFix'] += '\n'
            if field == 'base': data['baseGuard']['binarySha256'] = 'b' * 64
            with self.subTest(field=field), self.assertRaises(ValueError): additional.qualify(data)

    def test_vulnerable_legacy_package_or_source_rejected(self):
        for field, value in [('packages', 'github.com/docker/docker/daemon\n'),
                             ('packages', 'github.com/docker/docker/plugin\n'),
                             ('files', shared.NAMES + '\tmodule/extra.go\t' + 'a' * 64 + '\n')]:
            data = data_fixture(); data['source'][field] += value
            with self.subTest(value=value), self.assertRaises(ValueError): additional.qualify(data)

    def test_every_official_openpgp_package_and_subpackage_rejected(self):
        for package in ['golang.org/x/crypto/openpgp', 'golang.org/x/crypto/openpgp/armor',
                        'golang.org/x/crypto/openpgp/packet', 'golang.org/x/crypto/openpgp/clearsign',
                        'golang.org/x/crypto/openpgp/errors', 'golang.org/x/crypto/openpgp/elgamal',
                        'golang.org/x/crypto/openpgp/s2k', 'golang.org/x/crypto/openpgp/future']:
            data = data_fixture(); data['source']['packages'] += package + '\n'
            data['baseGuard'] = shared.source_check(data['source'])
            with self.subTest(package=package), self.assertRaises(ValueError): additional.qualify(data)

    def test_compiled_openpgp_file_or_module_version_rejected(self):
        data = data_fixture()
        data['source']['files'] += shared.MAIN + '\tmodule/golang.org/x/crypto@v0.57.0/openpgp/read.go\t' + 'a' * 64 + '\n'
        data['baseGuard'] = shared.source_check(data['source'])
        with self.assertRaises(ValueError): additional.qualify(data)
        data = data_fixture(); data['source']['modulesAfter'] = data['source']['modulesAfter'].replace('v0.57.0', 'v0.58.0')
        with self.assertRaises(ValueError): additional.qualify(data)

    def test_maintained_openpgp_fork_has_a_different_official_module_scope(self):
        data = data_fixture(); package = 'github.com/ProtonMail/go-crypto/openpgp'
        data['source']['packages'] += package + '\n'
        data['source']['files'] += package + '\tmodule/github.com/!proton!mail/go-crypto@v1.4.1/openpgp/read.go\t' + 'a' * 64 + '\n'
        data['baseGuard'] = shared.source_check(data['source'])
        additional.qualify(data)

    def test_fixed_target_exact_record_and_purl_only(self):
        data = data_fixture(); proof = additional.qualify(data)
        finding = json.loads(data['source']['raw'])['Results'][0]['Vulnerabilities'][-1]
        self.assertTrue(additional.matches(shared.TARGET, finding, proof))
        self.assertFalse(additional.matches('usr/bin/containerd', finding, proof))
        for field, value in [('VulnerabilityID', 'GO-UNKNOWN'), ('Severity', 'LOW'),
                             ('InstalledVersion', 'v0.58.0'), ('PkgIdentifier', {'PURL': shared.PURL})]:
            changed = dict(finding, **{field: value})
            with self.subTest(field=field): self.assertFalse(additional.matches(shared.TARGET, changed, proof))

    def test_existing_builder_policy_recomputes_proof_and_keeps_unknowns(self):
        data = data_fixture(); proof = additional.qualify(data); repo = 'PastureStack/node-agent'
        raw = json.loads(data['source']['raw']); result = copy.deepcopy(raw['Results'][0])
        result.update(Type='gobinary', Packages=[{'Name': 'example'}])
        result['Vulnerabilities'] = [v for v in result['Vulnerabilities'] if v['VulnerabilityID'] in additional.SCOPES]
        report = {'SchemaVersion': 2, 'Results': [result] + [
            {'Target': target, 'Type': 'gobinary', 'Packages': [{'Name': 'example'}]}
            for target in ['usr/local/go/bin/go'] + ['usr/bin/' + p for p in (
                'docker', 'dockerd', 'docker-proxy', 'containerd', 'containerd-shim-runc-v2', 'ctr', 'runc')]]}
        node = {'buildxInput': data, 'buildxQualification': proof, 'containerGraphs': {
            name: 'github.com/containerd/containerd/v2/cmd/' + name for name in (
                'containerd', 'ctr', 'containerd-shim-runc-v2')}}
        node['containerGraphs']['runc'] = 'github.com/opencontainers/runc'
        headers = {'repository': repo, 'package': 'linux-libc-dev', 'version': '7.0.0-38.38',
                   'purl': 'pkg:deb/ubuntu/linux-libc-dev@7.0.0-38.38?arch=amd64&distro=ubuntu-26.04', 'ids': []}
        run = lambda: policy.builder_violations(report, {'repository': repo, 'unfixed': []}, repo, {}, headers, node_proof=node)
        original = copy.deepcopy(report); self.assertEqual(run(), []); self.assertEqual(report, original)
        unknown = copy.deepcopy(result['Vulnerabilities'][0]); unknown['VulnerabilityID'] = 'CVE-UNKNOWN'
        result['Vulnerabilities'].append(unknown); self.assertTrue(run()); result['Vulnerabilities'].pop()
        node['buildxQualification']['binarySha256'] = 'tampered'
        with self.assertRaises(ValueError): run()
        node.pop('buildxInput'); self.assertTrue(run())


if __name__ == '__main__':
    unittest.main()
