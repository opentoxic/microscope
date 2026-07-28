module MicroscopeClient
  # Best-effort Ruby runtime metrics, shaped like every other microscope SDK:
  # a name, a language tag, a primary value + unit, plus language-specific extras.
  module RuntimeMetrics
    def self.sample
      stat = GC.stat
      {
        name: "ruby.runtime",
        language: "ruby",
        value: Thread.list.count,
        unit: "threads",
        threads: Thread.list.count,
        gc_count: GC.count,
        heap_live_slots: stat[:heap_live_slots],
      }
    end
  end
end
