from tests.platformcompat import Config, CONFIG_OVERRIDE


def test_compatibility_uuid_does_not_touch_configured_host_path(tmp_path):
    target = tmp_path / 'host-identity'
    first = Config._get_uuid_from_file(str(target))
    second = Config._get_uuid_from_file(str(target))
    assert first == second
    assert not target.exists()


def test_forced_compatibility_uuid_does_not_write_host_path(tmp_path):
    target = tmp_path / 'forced-host-identity'
    CONFIG_OVERRIDE['TEST_UUID'] = 'known-identity'
    try:
        result = Config.get_uuid_from_file('TEST_UUID', str(target),
                                           force_write=True)
    finally:
        del CONFIG_OVERRIDE['TEST_UUID']
    assert result == 'known-identity'
    assert not target.exists()
