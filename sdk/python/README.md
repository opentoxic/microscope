# microscope-client (Python)

Thin HTTP client for the [microscope](https://github.com/qobly/microscope) observability API.

## Install

```bash
pip install microscope-client
```

## Usage

```python
from microscope_client import MicroscopeClient

client = MicroscopeClient(base_url="http://localhost:8093/microscope")

client.record("payment_charged", content={"amount": 4200})
entries = client.list_entries(type="custom", limit=20)
entry = client.get_entry(entries["items"][0]["id"])
```

## Runtime metrics

Periodically records thread count, memory, and GC stats so the dashboard's metrics view has
something to show for Python services, the same way it does for Go:

```python
client.start_runtime_metrics(interval=15)  # call once at startup
```

## Django

```bash
pip install "microscope-client[django]"
```

```python
# settings.py
MICROSCOPE_BASE_URL = "http://localhost:8093/microscope"
MIDDLEWARE = [
    "microscope_client.integrations.django.MicroscopeMiddleware",
    ...
]
```

## FastAPI

```bash
pip install "microscope-client[fastapi]"
```

```python
from fastapi import FastAPI
from microscope_client.integrations.fastapi import MicroscopeMiddleware

app = FastAPI()
app.add_middleware(MicroscopeMiddleware, base_url="http://localhost:8093/microscope")
```

## Testing

```bash
pip install -e ".[test]"
python -m unittest discover -s tests -v
```
