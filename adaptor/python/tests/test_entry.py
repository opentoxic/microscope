from microscope.entry import ALL_ENTRY_TYPES, Entry


def test_entry_to_dict_omits_empty_optional_fields() -> None:
    entry = Entry(
        id="abc",
        batch_id="abc",
        type="request",
        content={"method": "GET"},
        created_at="2026-01-01T00:00:00Z",
    )
    assert entry.to_dict() == {
        "id": "abc",
        "batch_id": "abc",
        "type": "request",
        "content": {"method": "GET"},
        "created_at": "2026-01-01T00:00:00Z",
    }


def test_all_entry_types_contains_request_and_metric() -> None:
    assert "request" in ALL_ENTRY_TYPES
    assert "metric" in ALL_ENTRY_TYPES
