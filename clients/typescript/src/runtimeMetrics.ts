import { performance } from "node:perf_hooks";

export interface RuntimeMetricsContent {
  name: string;
  language: string;
  value: number;
  unit: string;
  [key: string]: unknown;
}

/** True when running under Node.js (not a browser). */
export function isNodeRuntime(): boolean {
  return typeof process !== "undefined" && !!process.versions?.node;
}

/**
 * Best-effort Node.js runtime metrics, shaped like every other microscope SDK:
 * a name, a language tag, a primary value + unit, plus language-specific extras.
 */
export function sampleRuntimeMetrics(): RuntimeMetricsContent {
  const memory = process.memoryUsage();
  const utilization = Math.round(performance.eventLoopUtilization().utilization * 100);

  return {
    name: "node.runtime",
    language: "node",
    value: utilization,
    unit: "%",
    event_loop_utilization_pct: utilization,
    memory_mb: memory.heapUsed / 1024 / 1024,
    rss_mb: memory.rss / 1024 / 1024,
  };
}
