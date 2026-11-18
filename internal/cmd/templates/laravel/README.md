# Mercy Pay dans votre app Laravel

Ces fichiers s'ajoutent à une app Laravel 12 ou 13 existante.

1. `composer require mercy-pay/laravel`
2. Copiez `app/`, `resources/` dans votre projet, et ajoutez la route de `routes/mercy-pay.php` à `routes/web.php`.
3. Ajoutez les lignes de `.env.example` à votre `.env`.
4. `php artisan serve`, puis ouvrez http://localhost:8000/paiement-test

Le paquet reçoit déjà les webhooks sur `POST /webhooks/mercy-pay`. En local :

```sh
mercypay listen --forward-to localhost:8000/webhooks/mercy-pay
```

Copiez le secret affiché dans `MERCY_PAY_WEBHOOK_SECRET`. Écoutez l'événement `MercyPay\Laravel\Events\WebhookReceived` pour marquer vos commandes payées.

Guide : https://docs.mercy-pay.com/payments/embedded
