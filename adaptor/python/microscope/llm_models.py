from __future__ import annotations

import base64
import json
import urllib.parse
import urllib.request
from typing import Any


def list_provider_models(provider: str, api_key: str) -> list[dict[str, str]]:
    provider = provider.strip().lower()
    if provider == "openai":
        return list_openai_models(api_key)
    if provider == "cursor":
        return list_cursor_models(api_key)
    if provider == "anthropic":
        return list_anthropic_models(api_key)
    if provider == "gemini":
        return list_gemini_models(api_key)
    raise ValueError(f"unsupported provider {provider!r}")


def call_llm(provider: str, model: str, api_key: str, prompt: str) -> str:
    provider = provider.strip().lower()
    if provider in {"openai", "cursor"}:
        endpoint = (
            "https://api.cursor.com/v1/chat/completions"
            if provider == "cursor"
            else "https://api.openai.com/v1/chat/completions"
        )
        payload: dict[str, Any] = {
            "model": model,
            "messages": [
                {"role": "system", "content": "Return only JSON. No markdown."},
                {"role": "user", "content": prompt},
            ],
        }
        if model_supports_custom_temperature(model):
            payload["temperature"] = 0.2
        return post_openai_compatible(endpoint, api_key, payload)
    if provider == "anthropic":
        return call_anthropic(model, api_key, prompt)
    if provider == "gemini":
        return call_gemini(model, api_key, prompt)
    raise ValueError(f"unsupported provider {provider!r}")


def model_supports_custom_temperature(model: str) -> bool:
    lower = model.lower()
    restricted = ("o1", "o3", "o4", "gpt-5", "reasoning", "thinking")
    return not any(token in lower for token in restricted)


def provider_get(endpoint: str, api_key: str, headers: dict[str, str] | None = None) -> tuple[bytes, int]:
    req = urllib.request.Request(endpoint)
    for key, value in (headers or {}).items():
        req.add_header(key, value)
    if api_key and not any(h in req.headers for h in ("Authorization", "x-api-key", "x-goog-api-key")):
        req.add_header("Authorization", f"Bearer {api_key}")
    with urllib.request.urlopen(req, timeout=25) as response:
        return response.read(4 * 1024 * 1024), response.status


def post_openai_compatible(endpoint: str, api_key: str, payload: dict[str, Any]) -> str:
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(endpoint, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("Authorization", f"Bearer {api_key}")
    try:
        with urllib.request.urlopen(req, timeout=45) as response:
            raw = response.read(1 * 1024 * 1024)
    except urllib.error.HTTPError as exc:
        raw = exc.read(1 * 1024 * 1024)
        if "temperature" in payload and "temperature" in raw.decode("utf-8", errors="replace"):
            del payload["temperature"]
            return post_openai_compatible(endpoint, api_key, payload)
        raise RuntimeError(f"provider returned {exc.code}: {raw.decode('utf-8', errors='replace').strip()}") from exc
    parsed = json.loads(raw)
    choices = parsed.get("choices", [])
    if not choices:
        raise RuntimeError("provider returned no choices")
    return str(choices[0].get("message", {}).get("content", ""))


def call_anthropic(model: str, api_key: str, prompt: str) -> str:
    payload = {
        "model": model,
        "max_tokens": 1800,
        "messages": [{"role": "user", "content": prompt}],
    }
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request("https://api.anthropic.com/v1/messages", data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("x-api-key", api_key)
    req.add_header("anthropic-version", "2023-06-01")
    with urllib.request.urlopen(req, timeout=45) as response:
        raw = response.read(1 * 1024 * 1024)
    parsed = json.loads(raw)
    content = parsed.get("content", [])
    if not content:
        raise RuntimeError("anthropic returned no content")
    return str(content[0].get("text", ""))


def call_gemini(model: str, api_key: str, prompt: str) -> str:
    endpoint = f"https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent"
    payload = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {"responseMimeType": "application/json"},
    }
    body = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(endpoint, data=body, method="POST")
    req.add_header("Content-Type", "application/json")
    req.add_header("x-goog-api-key", api_key)
    with urllib.request.urlopen(req, timeout=45) as response:
        raw = response.read(1 * 1024 * 1024)
    parsed = json.loads(raw)
    candidates = parsed.get("candidates", [])
    if not candidates or not candidates[0].get("content", {}).get("parts"):
        raise RuntimeError("gemini returned no candidates")
    return str(candidates[0]["content"]["parts"][0].get("text", ""))


def list_openai_models(api_key: str) -> list[dict[str, str]]:
    after = ""
    models: list[dict[str, str]] = []
    for _ in range(20):
        endpoint = "https://api.openai.com/v1/models"
        if after:
            endpoint += "?after=" + urllib.parse.quote(after)
        raw, status = provider_get(endpoint, api_key, {"Authorization": f"Bearer {api_key}"})
        if status >= 400:
            raise RuntimeError(f"openai returned {status}: {raw.decode('utf-8', errors='replace').strip()}")
        parsed = json.loads(raw)
        for item in parsed.get("data", []):
            model_id = item.get("id", "")
            if not openai_chat_model(model_id):
                continue
            created = ""
            if item.get("created"):
                from datetime import datetime, timezone

                created = datetime.fromtimestamp(int(item["created"]), tz=timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
            models.append({"id": model_id, "label": model_id, "created_at": created})
            after = model_id
        if not parsed.get("has_more") or not parsed.get("data"):
            break
    if not models:
        raise RuntimeError("openai returned no chat-capable models")
    return sort_provider_models(models)


def openai_chat_model(model_id: str) -> bool:
    lower = model_id.lower()
    excluded = (
        "embedding",
        "whisper",
        "tts",
        "dall-e",
        "davinci",
        "babbage",
        "moderation",
        "realtime",
        "audio",
        "transcribe",
        "search",
        "sora",
        "omni-moderation",
        "curie",
        "ada",
        "codex",
        "edit",
        "similarity",
        "instruct",
    )
    if any(token in lower for token in excluded):
        return False
    if lower.startswith(("gpt-", "o1", "o3", "o4", "chatgpt")):
        return True
    return "gpt" in lower or lower.startswith("o")


def list_cursor_models(api_key: str) -> list[dict[str, str]]:
    try:
        models = list_cursor_models_with_auth(api_key, basic=True)
        if models:
            return models
    except RuntimeError:
        pass
    models = list_cursor_models_with_auth(api_key, basic=False)
    if not models:
        raise RuntimeError("cursor returned no models")
    return models


def list_cursor_models_with_auth(api_key: str, basic: bool) -> list[dict[str, str]]:
    req = urllib.request.Request("https://api.cursor.com/v1/models")
    if basic:
        token = base64.b64encode(f"{api_key}:".encode("utf-8")).decode("ascii")
        req.add_header("Authorization", f"Basic {token}")
    else:
        req.add_header("Authorization", f"Bearer {api_key}")
    with urllib.request.urlopen(req, timeout=25) as response:
        raw = response.read(2 * 1024 * 1024)
    parsed = json.loads(raw)
    models: list[dict[str, str]] = []
    for item in parsed.get("items", []):
        model_id = str(item.get("id", "")).strip()
        if not model_id:
            continue
        label = str(item.get("displayName", "")).strip() or model_id
        models.append({"id": model_id, "label": label})
    return sort_provider_models(models)


def list_anthropic_models(api_key: str) -> list[dict[str, str]]:
    after_id = ""
    models: list[dict[str, str]] = []
    for _ in range(20):
        query = urllib.parse.urlencode({"limit": "1000", **({"after_id": after_id} if after_id else {})})
        endpoint = f"https://api.anthropic.com/v1/models?{query}"
        raw, status = provider_get(
            endpoint,
            api_key,
            {"x-api-key": api_key, "anthropic-version": "2023-06-01"},
        )
        if status >= 400:
            raise RuntimeError(f"anthropic returned {status}: {raw.decode('utf-8', errors='replace').strip()}")
        parsed = json.loads(raw)
        for item in parsed.get("data", []):
            label = str(item.get("display_name", "")).strip() or item.get("id", "")
            models.append(
                {
                    "id": item.get("id", ""),
                    "label": label,
                    "created_at": item.get("created_at", ""),
                }
            )
        if not parsed.get("has_more") or not parsed.get("last_id") or parsed.get("last_id") == after_id:
            break
        after_id = parsed.get("last_id", "")
    if not models:
        raise RuntimeError("anthropic returned no models")
    return sort_provider_models(models)


def list_gemini_models(api_key: str) -> list[dict[str, str]]:
    page_token = ""
    models: list[dict[str, str]] = []
    seen: set[str] = set()
    for _ in range(20):
        query = urllib.parse.urlencode({"pageSize": "1000", **({"pageToken": page_token} if page_token else {})})
        endpoint = f"https://generativelanguage.googleapis.com/v1beta/models?{query}"
        raw, status = provider_get(endpoint, api_key, {"x-goog-api-key": api_key})
        if status >= 400:
            raise RuntimeError(f"gemini returned {status}: {raw.decode('utf-8', errors='replace').strip()}")
        parsed = json.loads(raw)
        for item in parsed.get("models", []):
            if not gemini_chat_model(item.get("name", ""), item.get("supportedGenerationMethods", [])):
                continue
            model_id = str(item.get("name", "")).replace("models/", "")
            if item.get("baseModelId"):
                model_id = item["baseModelId"]
            if model_id in seen:
                continue
            seen.add(model_id)
            label = str(item.get("displayName", "")).strip() or model_id
            models.append({"id": model_id, "label": label})
        page_token = parsed.get("nextPageToken", "")
        if not page_token:
            break
    if not models:
        raise RuntimeError("gemini returned no generative models")
    return sort_provider_models(models)


def gemini_chat_model(name: str, methods: list[str]) -> bool:
    lower = name.lower()
    excluded = ("embedding", "aqa", "imagen", "veo", "tts", "lyria", "gemma")
    if any(token in lower for token in excluded):
        return False
    if not methods:
        return "gemini" in lower
    return any(method in {"generateContent", "countTokens"} for method in methods)


def sort_provider_models(models: list[dict[str, str]]) -> list[dict[str, str]]:
    return sorted(
        models,
        key=lambda item: (
            -(1 if item.get("created_at") else 0),
            item.get("created_at", ""),
            item.get("label", "").lower(),
        ),
    )
