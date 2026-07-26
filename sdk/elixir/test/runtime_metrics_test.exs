defmodule MicroscopeClient.RuntimeMetricsTest do
  use ExUnit.Case, async: true

  alias MicroscopeClient.RuntimeMetrics

  test "sample/0 returns the expected shape" do
    metrics = RuntimeMetrics.sample()

    assert metrics.name == "elixir.runtime"
    assert metrics.language == "elixir"
    assert metrics.unit == "processes"
    assert is_integer(metrics.value)
    assert metrics.value > 0
    assert is_integer(metrics.schedulers)
    assert metrics.schedulers > 0
    assert is_float(metrics.memory_mb)
    assert metrics.memory_mb > 0
  end
end
