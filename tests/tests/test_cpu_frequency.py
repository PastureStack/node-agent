from tests.platformcompat.plugins.host_info.cpu import ghz_to_mhz


def test_ghz_to_mhz_accepts_normal_model_name():
    assert ghz_to_mhz('Example CPU @ 3.40GHz') == 3400


def test_ghz_to_mhz_rejects_ambiguous_or_invalid_values():
    assert ghz_to_mhz('Example CPU without frequency') is None
    assert ghz_to_mhz('Example CPU @ invalidGHz') is None
    assert ghz_to_mhz(('9.' * 10000) + 'GHz') is None
