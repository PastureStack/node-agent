#!/usr/bin/env python3
"""Join a verified single-ELF native VEX result into the unchanged full raw report."""
import copy
import hashlib
import json
import sys
from collections import Counter

import check_buildx_code_absence as shared


def canonical(value):
    return json.dumps(value, sort_keys=True, separators=(',', ':'))


def transport_finding(finding, artifact_id, diff_ids, layered):
    value = copy.deepcopy(finding)
    # Trivy v0.74.0 pkg/fingerprint/fingerprint.go:46-52. A rootfs report has
    # no ArtifactID; image and rootfs therefore have different fingerprints.
    expected = 'sha256:' + hashlib.sha256(
        f"{artifact_id}:{shared.TARGET}:{value['PkgID']}:{value['VulnerabilityID']}".encode()
    ).hexdigest()
    shared.require(value.pop('Fingerprint', None) == expected, 'Invalid native transport fingerprint')
    layer = value.pop('Layer', None)
    if layered:
        shared.require(isinstance(layer, dict) and set(layer) == {'DiffID'}
                       and layer['DiffID'] in diff_ids, 'Buildx layer is not in the raw image')
    else:
        shared.require(layer is None, 'An isolated ELF unexpectedly has an image layer')
    return canonical(value)


def transport_packages(result, diff_ids, layered):
    packages = []
    for package in result.get('Packages', []):
        value = copy.deepcopy(package)
        layer = value.pop('Layer', None)
        if layered:
            shared.require(isinstance(layer, dict) and set(layer) == {'DiffID'}
                           and layer['DiffID'] in diff_ids, 'Package layer is not in the raw image')
        else:
            shared.require(layer is None, 'An isolated package unexpectedly has an image layer')
        packages.append(canonical(value))
    shared.require(packages, 'Buildx package inventory is missing')
    return Counter(packages)


def single_result(report):
    shared.require(report.get('ArtifactType') == 'filesystem'
                   and report.get('ArtifactName') == '/scan' and not report.get('ArtifactID'),
                   'Expected the fixed isolated ELF transport')
    results = report.get('Results', [])
    shared.require(len(results) == 1 and results[0].get('Target') == shared.TARGET
                   and results[0].get('Type') == 'gobinary' and results[0].get('Class') == 'lang-pkgs',
                   'The isolated scan contains another target')
    return results[0]


def join(data):
    raw = shared.load(data['raw'])
    isolated_raw = shared.load(data['isolatedRaw'])
    isolated_effective = shared.load(data['isolatedEffective'])
    guard, isolated_guard = data['guard'], data['isolatedGuard']
    shared.require(shared.vex_pairs(data['vex']) == {(cve, shared.PURL) for cve in shared.CVES},
                   'The isolated VEX exceeds this source-qualified module scope')
    shared.require(guard.get('status') == 'BUILDX_DAEMON_CODE_ABSENCE_VERIFIED'
                   and shared.sha(data['raw']) == guard['rawSha256'], 'Original raw binding changed')
    shared.require({k: v for k, v in guard.items() if k != 'rawSha256'}
                   == {k: v for k, v in isolated_guard.items() if k != 'rawSha256'},
                   'The isolated ELF has another binary or source qualification')
    native = shared.native_check({'raw': data['isolatedRaw'], 'effective': data['isolatedEffective'],
                                  'guard': isolated_guard, 'vex': data['vex']})
    original = single_result(isolated_raw)
    effective = single_result(isolated_effective)
    shared.require({k: v for k, v in original.items() if k not in ('Vulnerabilities', 'ModifiedFindings')}
                   == {k: v for k, v in effective.items() if k not in ('Vulnerabilities', 'ModifiedFindings')},
                   'Native VEX changed isolated package inventory or another result field')
    original_findings = Counter(transport_finding(v, '', set(), False)
                                for v in original.get('Vulnerabilities', []))
    retained = Counter(transport_finding(v, '', set(), False)
                       for v in effective.get('Vulnerabilities', []))
    packages = transport_packages(original, set(), False)
    shared.require(raw.get('ArtifactType') == 'container_image' and raw.get('ArtifactID'),
                   'Original full image identity is missing')
    diff_ids = set(raw.get('Metadata', {}).get('DiffIDs', []))
    shared.require(diff_ids, 'Original image layers are missing')
    scoped = [(t, v) for t, v in shared.findings(raw)
              if v.get('PkgIdentifier', {}).get('PURL') == shared.PURL]
    shared.require(scoped and all(t == shared.TARGET for t, _ in scoped),
                   'Another binary contains the qualified legacy PURL')
    output = copy.deepcopy(raw)
    matched = []
    for result in output['Results']:
        if result.get('Target') != shared.TARGET:
            continue
        shared.require(result.get('Type') == 'gobinary' and result.get('Class') == 'lang-pkgs',
                       'The full raw Buildx target changed type')
        values = result.get('Vulnerabilities', [])
        layered = bool(values and values[0].get('Layer'))
        artifact_id = raw['ArtifactID'] if layered else ''
        keys = [transport_finding(v, artifact_id, diff_ids, layered) for v in values]
        shared.require(Counter(keys) == original_findings, 'Full and isolated Buildx findings differ')
        shared.require(transport_packages(result, diff_ids, layered) == packages,
                       'Full and isolated Buildx package inventories differ')
        keep = retained.copy()
        selected = []
        for value, key in zip(values, keys):
            if keep[key]:
                selected.append(value)
                keep[key] -= 1
        shared.require(not +keep, 'A native retained finding could not be joined')
        result['Vulnerabilities'] = selected
        matched.append(layered)
    # The established full raw is image + standalone container-tools results.
    shared.require(sorted(matched) == [False, True], 'Expected both original Buildx scan transports')
    encoded = json.dumps(output)
    full_native = shared.native_check({'raw': data['raw'], 'effective': encoded,
                                       'guard': guard, 'vex': data['vex']})
    return {'effective': output, 'receipt': {
        'schemaVersion': 1, 'status': 'NATIVE_SINGLE_ELF_JOIN_VERIFIED',
        'target': shared.TARGET, 'binarySha256': guard['binarySha256'],
        'isolatedRawSha256': shared.sha(data['isolatedRaw']),
        'isolatedEffectiveSha256': shared.sha(data['isolatedEffective']),
        'isolatedNative': native, 'fullNative': full_native,
        'transportFieldsVerified': ['Layer.DiffID', 'Fingerprint'],
        'otherTargetsRetainedUnchanged': True}}


if __name__ == '__main__':
    try:
        content = sys.stdin.buffer.read(64 * 1024 * 1024 + 1)
        shared.require(len(content) <= 64 * 1024 * 1024, 'Evidence exceeds size limit')
        print(json.dumps(join(shared.load(content.decode('utf-8')))))
    except (ValueError, KeyError, TypeError, UnicodeError) as error:
        print('BUILDX_NATIVE_JOIN_FAILED: ' + str(error), file=sys.stderr)
        sys.exit(1)
