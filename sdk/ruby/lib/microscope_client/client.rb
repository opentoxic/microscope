module MicroscopeClient
  class Error < StandardError; end

  class Client
    def initialize(base_url:, timeout: 5)
      @base_url = base_url.chomp("/")
      @timeout = timeout
    end

    def record(name, content: {})
      response = post("/api/entries", { name: name, content: content })
      response.fetch("id")
    end

    def list_entries(type: nil, search: nil, limit: nil, offset: nil)
      params = { type: type, search: search, limit: limit, offset: offset }.compact
      get("/api/entries", params)
    end

    def get_entry(entry_id)
      get("/api/entries/#{entry_id}")
    end

    # Periodically records this process's runtime metrics (threads, GC) so
    # the dashboard's metrics view has something to show for Ruby services,
    # the same way it does for Go. Safe to call once at startup; a second
    # call is a no-op unless `stop_runtime_metrics` was called first.
    def start_runtime_metrics(interval: 15)
      return if @metrics_thread

      @metrics_stop = false
      @metrics_thread = Thread.new do
        until @metrics_stop
          begin
            metrics = RuntimeMetrics.sample
            record(metrics[:name], content: metrics)
          rescue StandardError
            nil
          end
          sleep(interval)
        end
      end
    end

    def stop_runtime_metrics
      @metrics_stop = true
      @metrics_thread&.wakeup rescue nil
      @metrics_thread = nil
    end

    private

    def post(path, body)
      uri = URI("#{@base_url}#{path}")
      request(Net::HTTP::Post.new(uri), uri, body)
    end

    def get(path, params = {})
      uri = URI("#{@base_url}#{path}")
      uri.query = URI.encode_www_form(params) unless params.empty?
      request(Net::HTTP::Get.new(uri), uri)
    end

    def request(req, uri, body = nil)
      req["content-type"] = "application/json"
      req.body = JSON.generate(body) if body

      http = Net::HTTP.new(uri.host, uri.port)
      http.use_ssl = uri.scheme == "https"
      http.read_timeout = @timeout

      response = http.request(req)
      unless response.is_a?(Net::HTTPSuccess)
        raise Error, "microscope: request failed with status #{response.code}"
      end

      JSON.parse(response.body)
    end
  end
end
