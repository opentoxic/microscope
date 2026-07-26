defmodule MicroscopeClientTest do
  use ExUnit.Case, async: true

  test "new/2 trims a trailing slash from base_url and applies default timeout" do
    client = MicroscopeClient.new("http://localhost:8093/microscope/")

    assert client.base_url == "http://localhost:8093/microscope"
    assert client.timeout == 5_000
  end

  test "new/2 accepts a custom timeout" do
    client = MicroscopeClient.new("http://localhost:8093/microscope", timeout: 1_000)

    assert client.timeout == 1_000
  end

  test "start_runtime_metrics/2 returns a running pid, stop_runtime_metrics/1 halts it" do
    # Points at a port nothing listens on: the reporter loop should keep
    # ticking (record/3 just returns {:error, _}) without crashing.
    client = MicroscopeClient.new("http://127.0.0.1:1/microscope")

    pid = MicroscopeClient.start_runtime_metrics(client, 20)
    assert Process.alive?(pid)

    :ok = MicroscopeClient.stop_runtime_metrics(pid)
    Process.sleep(10)
    refute Process.alive?(pid)
  end
end
