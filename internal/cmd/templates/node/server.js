// Une boutique minimale : une session de paiement intégrée, et les webhooks.
import express from 'express';
import { MercyPay, SignatureVerificationError } from '@mercy-pay/node';

const mercyPay = new MercyPay(process.env.MERCY_PAY_API_KEY);
const app = express();

// Le corps brut pour les webhooks : la signature porte sur lui.
app.post('/webhooks/mercy-pay', express.raw({ type: 'application/json' }), async (req, res) => {
  try {
    const event = await MercyPay.webhooks.constructEvent(req.body, req.get('MercyPay-Signature'), process.env.MERCY_PAY_WEBHOOK_SECRET);
    console.log('Webhook reçu :', event.type, event.data.id);
    res.sendStatus(200);
  } catch (error) {
    if (error instanceof SignatureVerificationError) return res.sendStatus(400);
    throw error;
  }
});

app.use(express.static('public'));

// La session se crée ici, côté serveur : la clé API ne va jamais dans le navigateur.
app.post('/session', async (_req, res) => {
  const session = await mercyPay.checkoutSessions.create(
    { amount: 1000, currency: 'XAF', description: 'Commande de test', ui_mode: 'embedded' },
    { idempotencyKey: `test-${Date.now()}` },
  );
  res.json({ client_secret: session.client_secret });
});

app.listen(4242, () => console.log('Boutique de test : http://localhost:4242'));
