# [[.Name]]

Une boutique Next.js (App Router) avec le paiement intégré Mercy Pay.

```sh
npm install
npm run dev
```

Ouvrez http://localhost:3000 et payez avec un numéro de test.

Pour les webhooks, dans un autre terminal :

```sh
mercypay listen --forward-to localhost:3000/api/webhooks/mercy-pay
```

Copiez le secret qu'il affiche dans `MERCY_PAY_WEBHOOK_SECRET` (fichier `.env.local`), puis relancez `npm run dev`.

Guide : https://docs.mercy-pay.com/payments/embedded
