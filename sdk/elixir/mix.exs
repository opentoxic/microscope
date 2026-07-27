defmodule MicroscopeClient.MixProject do
  use Mix.Project

  def project do
    [
      app: :microscope_client,
      version: "0.1.0",
      elixir: "~> 1.15",
      description: "Thin HTTP client for the microscope observability API",
      package: package(),
      deps: deps()
    ]
  end

  def application do
    [extra_applications: [:logger, :inets, :ssl]]
  end

  defp deps do
    [
      {:jason, "~> 1.4"}
    ]
  end

  defp package do
    [
      licenses: ["MIT"],
      links: %{"GitHub" => "https://github.com/opentoxic/microscope"}
    ]
  end
end
