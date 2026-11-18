# [[.Name]]

Une boutique Node.js minimale avec le paiement intégré Mercy Pay.

```sh
npm install
npm start
```

Ouvrez http://localhost:4242 et payez avec un numéro de test (les boutons de test s'affichent en sandbox).

Pour les webhooks, dans un autre terminal :

```sh
mercypay listen --forward-to localhost:4242/webhooks/mercy-pay
```

Copiez le secret qu'il affiche dans `MERCY_PAY_WEBHOOK_SECRET` (fichier `.env`), puis relancez `npm start`.

Guide : https://docs.mercy-pay.com/payments/embedded
