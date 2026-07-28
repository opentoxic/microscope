from django.urls import path, re_path

from microscope_django.views import api_view, spa_view, stream_view

urlpatterns = [
    path("api/stream", stream_view, name="microscope-stream"),
    re_path(r"^api/.*$", api_view, name="microscope-api"),
    re_path(r"^.*$", spa_view, name="microscope-spa"),
]
