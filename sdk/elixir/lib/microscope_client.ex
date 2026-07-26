defmodule MicroscopeClient do
  @moduledoc """
  Thin HTTP client for the microscope observability API.
  """

  defstruct [:base_url, timeout: 5_000]

  @type t :: %__MODULE__{base_url: String.t(), timeout: non_neg_integer()}

  @spec new(String.t(), keyword()) :: t()
  def new(base_url, opts \\ []) do
    %__MODULE__{
      base_url: String.trim_trailing(base_url, "/"),
      timeout: Keyword.get(opts, :timeout, 5_000)
    }
  end

  @spec record(t(), String.t(), map()) :: {:ok, String.t()} | {:error, term()}
  def record(%__MODULE__{} = client, name, content \\ %{}) do
    with {:ok, body} <- post(client, "/api/entries", %{name: name, content: content}) do
      {:ok, Map.fetch!(body, "id")}
    end
  end

  @spec list_entries(t(), keyword()) :: {:ok, map()} | {:error, term()}
  def list_entries(%__MODULE__{} = client, opts \\ []) do
    query =
      opts
      |> Keyword.take([:type, :search, :limit, :offset])
      |> Enum.reject(fn {_, v} -> is_nil(v) end)
      |> URI.encode_query()

    path = if query == "", do: "/api/entries", else: "/api/entries?" <> query
    get(client, path)
  end

  @spec get_entry(t(), String.t()) :: {:ok, map()} | {:error, term()}
  def get_entry(%__MODULE__{} = client, entry_id) do
    get(client, "/api/entries/" <> URI.encode(entry_id))
  end

  @doc """
  Periodically records this node's runtime metrics (process count,
  schedulers, memory) so the dashboard's metrics view has something to show
  for Elixir services, the same way it does for Go. Returns the reporter
  pid; stop it with `stop_runtime_metrics/1`.
  """
  @spec start_runtime_metrics(t(), non_neg_integer()) :: pid()
  def start_runtime_metrics(%__MODULE__{} = client, interval \\ 15_000) do
    spawn(fn -> runtime_metrics_loop(client, interval) end)
  end

  @spec stop_runtime_metrics(pid()) :: :ok
  def stop_runtime_metrics(pid) do
    Process.exit(pid, :normal)
    :ok
  end

  defp runtime_metrics_loop(client, interval) do
    metrics = MicroscopeClient.RuntimeMetrics.sample()
    record(client, metrics.name, metrics)
    Process.sleep(interval)
    runtime_metrics_loop(client, interval)
  end

  defp post(client, path, body) do
    request(client, :post, path, Jason.encode!(body))
  end

  defp get(client, path) do
    request(client, :get, path, nil)
  end

  defp request(%__MODULE__{base_url: base_url, timeout: timeout}, method, path, body) do
    url = String.to_charlist(base_url <> path)
    headers = [{~c"accept", ~c"application/json"}]
    http_opts = [timeout: timeout]

    request =
      case {method, body} do
        {:get, _} -> {url, headers}
        {:post, body} -> {url, [{~c"content-type", ~c"application/json"} | headers], ~c"application/json", body}
      end

    case :httpc.request(method, request, http_opts, []) do
      {:ok, {{_, status, _}, _headers, resp_body}} when status in 200..299 ->
        {:ok, Jason.decode!(IO.iodata_to_binary(resp_body))}

      {:ok, {{_, status, _}, _headers, _resp_body}} ->
        {:error, {:unexpected_status, status}}

      {:error, reason} ->
        {:error, reason}
    end
  end
end
