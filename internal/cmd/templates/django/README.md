# Mercy Pay dans votre projet Django

Cette app s'ajoute à un projet Django 4.2 ou plus.

1. `pip install mercy-pay-django`
2. Copiez le dossier `mercypay_demo/` dans votre projet, et ajoutez `"mercypay_django"` et `"mercypay_demo"` à `INSTALLED_APPS`.
3. Dans `urls.py` : `path("", include("mercypay_demo.urls"))`.
4. Exportez les variables de `.env.example` (`MERCY_PAY_API_KEY`, `MERCY_PAY_WEBHOOK_SECRET`).
5. `python manage.py runserver`, puis ouvrez http://localhost:8000/paiement-test/

En local, les webhooks arrivent avec :

```sh
mercypay listen --forward-to localhost:8000/webhooks/mercy-pay/
```

Guide : https://docs.mercy-pay.com/payments/embedded
