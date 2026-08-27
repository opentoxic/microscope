from datetime import datetime, timezone

from microscope.batch_group import BatchTypeGroup, group_batch_by_type
from microscope.detail import (
    build_entry_detail,
    entry_content_tabs,
    group_batch_related,
    looks_like_json,
)
from microscope.entry import Entry


def test_group_batch_related_excludes_current() -> None:
    now = "2026-01-01T00:00:00Z"
    batch = [
        Entry("e1", "b1", "request", {}, now),
        Entry("e2", "b1", "query", {}, now),
        Entry("e3", "b1", "log", {}, now),
    ]
    groups = group_batch_related(batch, "e1")
    assert len(groups) == 2
    assert groups[0].type == "query"


def test_entry_content_tabs_request() -> None:
    tabs = entry_content_tabs(
        Entry(
            "e1",
            "b1",
            "request",
            {
                "headers": {"Accept": ["application/json"]},
                "response_body": '{"status":"ready"}',
            },
            "2026-01-01T00:00:00Z",
        )
    )
    assert len(tabs) == 2


def test_looks_like_json_requires_valid_document() -> None:
    assert looks_like_json('{"status":"ready","items":[1,2]}')
    assert looks_like_json("[{\"id\":1}]")
    assert not looks_like_json('{"status":')


def test_batch_group_summary_queries() -> None:
    now = "2026-01-01T00:00:00Z"
    from microscope.detail import batch_group_summary

    summary = batch_group_summary(
        BatchTypeGroup(
            type="query",
            label="Queries",
            entries=[
                Entry("1", "b", "query", {"sql": "SELECT 1"}, now),
                Entry("2", "b", "query", {"sql": "SELECT 1"}, now),
            ],
        )
    )
    assert summary == "2 queries, 1 of which are duplicated."


def test_group_batch_by_type() -> None:
    now = "2026-01-01T00:00:00Z"
    batch = [
        Entry("1", "b", "query", {}, now),
        Entry("2", "b", "request", {}, now),
        Entry("3", "b", "query", {}, now),
        Entry("4", "b", "log", {}, now),
    ]
    groups = group_batch_by_type(batch)
    assert len(groups) == 3
    assert groups[0].type == "request"
    assert groups[1].type == "query"
    assert groups[2].type == "log"


def test_build_entry_detail_shape() -> None:
    now = "2026-01-01T00:00:00Z"
    entry = Entry("e1", "b1", "request", {"path": "/health"}, now)
    batch = [
        entry,
        Entry("e2", "b1", "query", {"sql": "SELECT 1"}, now),
    ]
    detail = build_entry_detail(entry, batch)
    assert detail["entry"]["id"] == "e1"
    assert len(detail["content_tabs"]) > 0
    assert len(detail["batch_groups"]) == 1
