from django.urls import include, path

from . import views

urlpatterns = [
    path("paiement-test/", views.checkout),
    path("webhooks/mercy-pay/", include("mercypay_django.urls")),
]
