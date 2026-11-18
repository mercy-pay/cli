# [[.Name]]

Une boutique PHP minimale avec le paiement intégré Mercy Pay.

```sh
composer install
php -S localhost:4242 -t public
```

Ouvrez http://localhost:4242 et payez avec un numéro de test.

Pour les webhooks, dans un autre terminal :

```sh
mercypay listen --forward-to localhost:4242/webhook.php
```

Copiez le secret qu'il affiche dans `MERCY_PAY_WEBHOOK_SECRET` (fichier `.env`).

Guide : https://docs.mercy-pay.com/payments/embedded
