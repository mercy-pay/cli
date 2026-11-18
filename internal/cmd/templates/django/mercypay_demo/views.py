from django.dispatch import receiver
from django.shortcuts import render
from mercypay_django import get_client
from mercypay_django.signals import webhook_received


def checkout(request):
    """Une commande de test payée avec le paiement intégré."""
    session = get_client().checkout_sessions.create(
        amount=1000,
        currency="XAF",
        description="Commande de test",
        ui_mode="embedded",
    )
    return render(request, "mercypay_demo/checkout.html", {"client_secret": session["client_secret"]})


@receiver(webhook_received)
def on_webhook(sender, event, **kwargs):
    print("Webhook reçu :", event["type"])
