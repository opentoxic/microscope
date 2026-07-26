require "minitest/autorun"
require_relative "../lib/microscope_client"

class RuntimeMetricsTest < Minitest::Test
  def test_sample_returns_expected_shape
    metrics = MicroscopeClient::RuntimeMetrics.sample

    assert_equal "ruby.runtime", metrics[:name]
    assert_equal "ruby", metrics[:language]
    assert_equal "threads", metrics[:unit]
    assert_kind_of Integer, metrics[:value]
    assert metrics[:value] >= 1
    assert_kind_of Integer, metrics[:gc_count]
    assert_kind_of Integer, metrics[:heap_live_slots]
  end
end
