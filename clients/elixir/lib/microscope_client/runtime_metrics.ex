defmodule MicroscopeClient.RuntimeMetrics do
  @moduledoc """
  Best-effort BEAM runtime metrics, shaped like every other microscope SDK: a
  name, a language tag, a primary value + unit, plus language-specific extras.
  """

  @spec sample() :: map()
  def sample do
    process_count = :erlang.system_info(:process_count)

    %{
      name: "elixir.runtime",
      language: "elixir",
      value: process_count,
      unit: "processes",
      process_count: process_count,
      schedulers: :erlang.system_info(:schedulers_online),
      memory_mb: :erlang.memory(:total) / 1_048_576
    }
  end
end
