#!/usr/bin/env python3
"""Qualify three additional Buildx source scopes; do not alter native VEX/raw."""
import json
import sys

import check_buildx_code_absence as shared

INPUT_SHAS = {
    'GO-2026-4883': 'a53fcda63a402fa87ab624163d9308d7e4b5e70eaee694e6fc594cced4ece327',
    'GO-2026-5668': 'be8fd95648c391f93ba3dfbef0fd245bbd73a44ab6e31dd325c4f1a922f6aa2b',
    'GO-2026-5932': 'f277b0400996200a7d5034c676661cd8cf18adeab1fdc5d0dc369e456ddf1fad',
}
PATCH_SHA = '50896e0a5d94c3586c16997cdfd6dc94214a511f7c1493449928d2f34f5e5793'
CRYPTO_PURL = 'pkg:golang/golang.org/x/crypto@v0.57.0'
SCOPES = {
    'CVE-2026-33997': ('github.com/docker/docker', 'v28.5.2+incompatible', shared.PURL),
    'CVE-2026-41568': ('github.com/docker/docker', 'v28.5.2+incompatible', shared.PURL),
    'GO-2026-5932': ('golang.org/x/crypto', 'v0.57.0', CRYPTO_PURL),
}


def record_sha(finding):
    return shared.sha(json.dumps(finding, sort_keys=True, separators=(',', ':')))


def qualify(data):
    # Reuse the complete original four-ID verifier, including binary/source,
    # actual compiled file SHA inventory and native source scanner bindings.
    source = data['source']
    base = shared.source_check(source)
    shared.require(base == data['baseGuard'], 'Original source qualification changed')
    shared.require(set(data['osv']) == set(INPUT_SHAS), 'Additional canonical inputs missing or unbounded')
    canonical = {}
    for identifier, digest in INPUT_SHAS.items():
        content = data['osv'][identifier]
        shared.require(shared.sha(content) == digest, 'Additional canonical input bytes changed')
        canonical[identifier] = shared.load(content)
        shared.require(canonical[identifier]['id'] == identifier, 'Wrong canonical advisory ID')
    shared.require(shared.sha(data['pluginFix']) == PATCH_SHA, 'Official plugin fix bytes changed')
    patch = shared.load(data['pluginFix'])
    patch_paths = ['daemon/pkg/plugin/manager.go', 'daemon/pkg/plugin/manager_test.go']
    shared.require(patch['sha'] == 'f4d6f25bf0c3fa12d4968320a45685947756a22a'
                   and sorted(f['filename'] for f in patch['files']) == patch_paths,
                   'Official plugin fix/package boundary changed')
    packages = source['packages'].splitlines()
    files = [line.split('\t') for line in source['files'].splitlines()]
    # The common verifier already requires the only legacy compiled package/file
    # to be the official namesgenerator. These two IDs affect plugin/daemon code.
    shared.require(not any('/plugin/' in path or path.endswith('/plugin')
                           for package, path, _ in files if package.startswith('github.com/docker/docker/')),
                   'Plugin source is compiled')
    crypto = canonical['GO-2026-5932']['affected'][0]
    affected = [p['path'] for p in crypto['ecosystem_specific']['imports']]
    shared.require(crypto['package']['name'] == 'golang.org/x/crypto' and len(affected) == 7
                   and all(p == 'golang.org/x/crypto/openpgp' or p.startswith('golang.org/x/crypto/openpgp/')
                           for p in affected), 'OpenPGP official package boundary changed')
    shared.require(not any(p == a or p.startswith(a + '/') for p in packages for a in affected)
                   and not any(package == 'golang.org/x/crypto/openpgp'
                               or package.startswith('golang.org/x/crypto/openpgp/')
                               or path.startswith('module/golang.org/x/crypto@v0.57.0/openpgp/')
                               for package, path, _ in files), 'An affected OpenPGP package/file is compiled')
    shared.require('golang.org/x/crypto v0.57.0' in source['modulesAfter'].splitlines(),
                   'Actual x/crypto module version changed')
    findings = []
    for target, finding in shared.findings(shared.load(source['raw'])):
        identifier = finding.get('VulnerabilityID')
        if target != shared.TARGET or identifier not in SCOPES:
            continue
        actual = (finding.get('PkgName'), finding.get('InstalledVersion'),
                  finding.get('PkgIdentifier', {}).get('PURL'))
        shared.require(actual == SCOPES[identifier], 'Additional finding package identity changed')
        findings.append({'id': identifier, 'findingSha256': record_sha(finding)})
    shared.require({f['id'] for f in findings} == set(SCOPES), 'Original additional finding controls missing')
    return {'schemaVersion': 1, 'status': 'BUILDX_ADDITIONAL_SOURCE_SCOPES_QUALIFIED',
            'target': shared.TARGET, 'binarySha256': base['binarySha256'], 'sourceGuard': base,
            'canonicalOsvSha256': dict(INPUT_SHAS), 'officialPluginFixSha256': PATCH_SHA,
            'officialPluginFixPaths': patch_paths, 'openpgpAffectedPackagesAbsent': affected,
            'notAffected': [{'id': identifier, 'package': value[0], 'version': value[1], 'purl': value[2],
                             'status': 'not_affected', 'justification': 'vulnerable_code_not_present'}
                            for identifier, value in SCOPES.items()],
            'qualifiedOriginalFindings': findings,
            'coverage': 'Three per-ID compiled-source determinations; original native govulncheck covers only its four pinned IDs. No claim of additional native filtering or a fixed dependency.'}


def matches(target, finding, qualification):
    return (target == shared.TARGET and finding.get('VulnerabilityID') in SCOPES
            and (finding.get('PkgName'), finding.get('InstalledVersion'),
                 finding.get('PkgIdentifier', {}).get('PURL')) == SCOPES[finding['VulnerabilityID']]
            and {'id': finding['VulnerabilityID'], 'findingSha256': record_sha(finding)}
            in qualification['qualifiedOriginalFindings'])


if __name__ == '__main__':
    try:
        content = sys.stdin.buffer.read(64 * 1024 * 1024 + 1)
        shared.require(len(content) <= 64 * 1024 * 1024, 'Evidence exceeds size limit')
        print(json.dumps(qualify(shared.load(content.decode('utf-8'))), sort_keys=True))
    except (ValueError, KeyError, TypeError, UnicodeError) as error:
        print('BUILDX_ADDITIONAL_SCOPE_FAILED: ' + str(error), file=sys.stderr)
        sys.exit(1)
