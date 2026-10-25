#!/usr/bin/env python3
"""Single-ELF join controls; all unrelated complete records stay immutable."""
import copy
import hashlib
import json
import unittest

import check_buildx_code_absence as shared
import join_buildx_native_vex as subject


IMAGE_ID = 'sha256:' + 'a' * 64
LAYER = 'sha256:' + 'b' * 64


def fingerprint(finding, artifact=''):
    finding['Fingerprint'] = 'sha256:' + hashlib.sha256(
        f"{artifact}:{shared.TARGET}:{finding['PkgID']}:{finding['VulnerabilityID']}".encode()
    ).hexdigest()


def finding(identifier):
    result = {'VulnerabilityID': identifier, 'PkgName': 'github.com/docker/docker',
              'PkgID': 'github.com/docker/docker@v28.5.2+incompatible',
              'InstalledVersion': 'v28.5.2+incompatible', 'FixedVersion': 'v99.0.0',
              'PkgIdentifier': {'PURL': shared.PURL, 'UID': 'same-binary-package'}, 'Severity': 'HIGH'}
    fingerprint(result)
    return result


def fixture():
    result = {'Target': shared.TARGET, 'Type': 'gobinary', 'Class': 'lang-pkgs',
              'Packages': [{'Name': 'example', 'Version': '1', 'Identifier': {'UID': 'same'}}],
              'Vulnerabilities': [finding(c) for c in sorted(shared.CVES)] + [finding('CVE-UNKNOWN')]}
    raw_elf = {'ArtifactName': '/scan', 'ArtifactType': 'filesystem', 'Results': [result]}
    effective = copy.deepcopy(raw_elf)
    effective['Results'][0]['Vulnerabilities'] = [copy.deepcopy(result['Vulnerabilities'][-1])]
    image = copy.deepcopy(result)
    for value in image['Vulnerabilities']:
        value['Layer'] = {'DiffID': LAYER}; fingerprint(value, IMAGE_ID)
    image['Packages'][0]['Layer'] = {'DiffID': LAYER}
    unrelated = {'Target': 'Python', 'Vulnerabilities': [{'VulnerabilityID': 'CVE-OTHER',
                  'PkgName': 'setuptools', 'PkgIdentifier': {'UID': 'original-python-uid',
                                                          'PURL': 'pkg:pypi/setuptools@70.3.0'},
                  'Layer': {'DiffID': 'original-python-layer'}, 'Severity': 'HIGH'}]}
    raw = {'ArtifactType': 'container_image', 'ArtifactID': IMAGE_ID,
           'Metadata': {'DiffIDs': [LAYER]}, 'Results': [image, unrelated, copy.deepcopy(result)]}
    vex = {'statements': [{'vulnerability': {'name': c}, 'products': [{'@id': shared.PURL}],
                           'status': 'not_affected', 'justification': 'vulnerable_code_not_present'}
                          for c in sorted(shared.CVES)]}
    base_guard = {'status': 'BUILDX_DAEMON_CODE_ABSENCE_VERIFIED', 'binarySha256': 'c' * 64,
                  'vexSha256': shared.sha(json.dumps(vex, sort_keys=True))}
    data = {'raw': json.dumps(raw), 'isolatedRaw': json.dumps(raw_elf),
            'isolatedEffective': json.dumps(effective), 'vex': vex,
            'guard': dict(base_guard), 'isolatedGuard': dict(base_guard)}
    rebind(data)
    return data


def rebind(data):
    data['guard']['rawSha256'] = shared.sha(data['raw'])
    data['isolatedGuard']['rawSha256'] = shared.sha(data['isolatedRaw'])


def change(data, field, mutate):
    value = json.loads(data[field]); mutate(value); data[field] = json.dumps(value)
    rebind(data)


class NativeJoinTests(unittest.TestCase):
    def test_join_preserves_original_layers_uids_and_unknown_findings(self):
        data = fixture(); original = json.loads(data['raw']); output = subject.join(data)
        self.assertEqual(output['effective']['Results'][1], original['Results'][1])
        self.assertEqual(output['effective']['Results'][0]['Vulnerabilities'],
                         original['Results'][0]['Vulnerabilities'][-1:])
        self.assertEqual(output['effective']['Results'][2]['Vulnerabilities'],
                         original['Results'][2]['Vulnerabilities'][-1:])
        self.assertTrue(output['receipt']['fullNative']['rawRetainedUnchanged'])

    def test_source_and_raw_bindings_are_required(self):
        for field in ['raw', 'binary', 'vex']:
            data = fixture()
            if field == 'raw': data['raw'] += '\n'
            if field == 'binary': data['isolatedGuard']['binarySha256'] = 'd' * 64
            if field == 'vex': data['vex']['statements'].pop()
            with self.subTest(field=field), self.assertRaises(ValueError): subject.join(data)

    def test_unrelated_finding_fields_cannot_be_normalized(self):
        for key, value in [('Severity', 'LOW'), ('FixedVersion', ''), ('InstalledVersion', 'different'),
                           ('PkgIdentifier', {'PURL': shared.PURL, 'UID': 'wrong'})]:
            data = fixture()
            change(data, 'raw', lambda r: r['Results'][0]['Vulnerabilities'][0].update({key: value}))
            with self.subTest(key=key), self.assertRaises(ValueError): subject.join(data)

    def test_invalid_transport_fingerprints_are_rejected(self):
        for field in ['raw', 'isolatedRaw', 'isolatedEffective']:
            data = fixture()
            change(data, field, lambda r: r['Results'][0]['Vulnerabilities'][0].update({'Fingerprint': 'forged'}))
            with self.subTest(field=field), self.assertRaises(ValueError): subject.join(data)

    def test_unverified_or_extra_layer_fields_are_rejected(self):
        for layer in [{'DiffID': 'unknown'}, {'DiffID': LAYER, 'Digest': 'extra'}, None]:
            data = fixture()
            change(data, 'raw', lambda r: r['Results'][0]['Vulnerabilities'][0].update({'Layer': layer}))
            with self.subTest(layer=layer), self.assertRaises(ValueError): subject.join(data)

    def test_isolated_scan_must_have_only_the_exact_binary(self):
        for field in ['isolatedRaw', 'isolatedEffective']:
            data = fixture(); change(data, field, lambda r: r['Results'].append({'Target': 'usr/bin/other'}))
            with self.subTest(field=field), self.assertRaises(ValueError): subject.join(data)
        data = fixture(); change(data, 'isolatedRaw', lambda r: r.update({'ArtifactID': IMAGE_ID}))
        with self.assertRaises(ValueError): subject.join(data)

    def test_package_inventory_and_uid_changes_are_rejected(self):
        for field in ['raw', 'isolatedEffective']:
            data = fixture()
            change(data, field, lambda r: r['Results'][0]['Packages'][0].update({'Identifier': {'UID': 'wrong'}}))
            with self.subTest(field=field), self.assertRaises(ValueError): subject.join(data)

    def test_unknown_native_finding_removal_is_rejected(self):
        data = fixture()
        change(data, 'isolatedEffective', lambda r: r['Results'][0].update({'Vulnerabilities': []}))
        with self.assertRaises(ValueError): subject.join(data)

    def test_qualified_native_finding_retained_is_rejected(self):
        data = fixture(); data['isolatedEffective'] = data['isolatedRaw']
        with self.assertRaises(ValueError): subject.join(data)

    def test_same_purl_in_another_binary_is_rejected(self):
        data = fixture()
        change(data, 'raw', lambda r: r['Results'].append({'Target': 'usr/bin/other',
                                                        'Vulnerabilities': [finding('CVE-UNKNOWN')]}))
        with self.assertRaises(ValueError): subject.join(data)

    def test_missing_or_duplicate_transport_is_rejected(self):
        for action in ['missing', 'duplicate']:
            data = fixture()
            change(data, 'raw', lambda r: r['Results'].pop() if action == 'missing'
                   else r['Results'].append(copy.deepcopy(r['Results'][0])))
            with self.subTest(action=action), self.assertRaises(ValueError): subject.join(data)


if __name__ == '__main__':
    unittest.main()
