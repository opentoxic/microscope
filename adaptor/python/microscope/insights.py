from __future__ import annotations

import json
from typing import Any

from microscope.llm_models import call_llm


def run_insight_analysis(input_data: dict[str, Any]) -> dict[str, Any]:
    prompt = build_insight_prompt(input_data)
    raw = call_llm(
        input_data.get("provider", ""),
        input_data.get("model", ""),
        input_data.get("api_key", ""),
        prompt,
    )
    return parse_insight_response(raw, input_data.get("entries", []))


def build_insight_prompt(input_data: dict[str, Any]) -> str:
    entries = input_data.get("entries", [])
    lines = [
        "You are an observability analyst for a Python/Django runtime recorder called Microscope.",
        "Analyze the telemetry and return ONLY valid JSON with this exact schema:",
        '{"summary":"string","health_score":0-100,"findings":[{"title":"string","detail":"string","severity":"info|warning|critical"}],"recommendations":["string"],"metrics":{"error_rate":"string","avg_latency_ms":"number","dominant_signal":"string","risk_level":"string"},"signal_distribution":[{"type":"string","count":number,"pct":number}]}',
    ]
    context = input_data.get("context", "")
    if context:
        lines.append(f"Context: {context}")
    period = input_data.get("period", "")
    if period:
        lines.append(f"Time window: {period}")
    lines.append(f"Entry count: {len(entries)}")
    lines.append("Entries:")
    for entry in entries:
        content = json.dumps(entry.get("content", {}))
        if len(content) > 900:
            content = content[:900]
        lines.append(
            f"- id={entry.get('id')} type={entry.get('type')} at={entry.get('created_at')} content={content}"
        )
    return "\n".join(lines) + "\n"


def parse_insight_response(raw: str, entries: list[dict[str, Any]]) -> dict[str, Any]:
    raw = raw.strip()
    start = raw.find("{")
    end = raw.rfind("}")
    if start >= 0 and end > start:
        raw = raw[start : end + 1]
    try:
        parsed = json.loads(raw)
    except json.JSONDecodeError:
        return fallback_insight(entries)
    score = int(parsed.get("health_score", 0))
    parsed["health_score"] = max(0, min(100, score))
    if not parsed.get("signal_distribution"):
        parsed["signal_distribution"] = distribution_for(entries)
    if parsed.get("metrics") is None:
        parsed["metrics"] = {}
    return parsed


def fallback_insight(entries: list[dict[str, Any]]) -> dict[str, Any]:
    errors = 0
    total_latency = 0.0
    latency_count = 0
    for entry in entries:
        if entry.get("type") == "exception":
            errors += 1
        content = entry.get("content", {})
        status = content.get("status")
        if isinstance(status, (int, float)) and status >= 500:
            errors += 1
        duration = content.get("duration_ms")
        if isinstance(duration, (int, float)) and duration > 0:
            total_latency += float(duration)
            latency_count += 1
    avg = total_latency / latency_count if latency_count else 0.0
    score = 92
    if errors > 0:
        score = 58
    elif avg >= 500:
        score = 72
    findings = [
        {
            "title": "Microscope coverage",
            "detail": f"{len(entries)} correlated records were included in this manual analysis window.",
            "severity": "info",
        }
    ]
    if errors > 0:
        findings.append(
            {
                "title": "Failure evidence detected",
                "detail": f"{errors} error-class records were present in the submitted window.",
                "severity": "critical",
            }
        )
    if avg >= 200:
        findings.append(
            {
                "title": "Latency pressure",
                "detail": f"Average recorded span cost is {avg:.0f}ms across timed operations.",
                "severity": "warning",
            }
        )
    recs = ["Inspect the dominant slow span first", "Compare this window against a healthy baseline trace"]
    if errors > 0:
        recs = [
            "Open the earliest failure boundary",
            "Correlate exceptions with request and SQL evidence",
            "Validate downstream dependency health",
        ]
    return {
        "summary": (
            f"Manual analysis across {len(entries)} records shows {errors} error-class events "
            f"and {avg:.0f}ms average span cost."
        ),
        "health_score": score,
        "findings": findings,
        "recommendations": recs,
        "metrics": {
            "error_rate": str(errors),
            "avg_latency_ms": avg,
            "dominant_signal": dominant_type(entries),
            "risk_level": risk_level(score),
        },
        "signal_distribution": distribution_for(entries),
    }


def distribution_for(entries: list[dict[str, Any]]) -> list[dict[str, Any]]:
    counts: dict[str, int] = {}
    for entry in entries:
        entry_type = str(entry.get("type", ""))
        counts[entry_type] = counts.get(entry_type, 0) + 1
    total = len(entries)
    if total == 0:
        return []
    return [
        {"type": typ, "count": count, "pct": count / total * 100}
        for typ, count in counts.items()
    ]


def dominant_type(entries: list[dict[str, Any]]) -> str:
    counts: dict[str, int] = {}
    for entry in entries:
        entry_type = str(entry.get("type", ""))
        counts[entry_type] = counts.get(entry_type, 0) + 1
    best = ""
    best_count = 0
    for typ, count in counts.items():
        if count > best_count:
            best = typ
            best_count = count
    return best


def risk_level(score: int) -> str:
    if score >= 85:
        return "low"
    if score >= 65:
        return "medium"
    return "high"
