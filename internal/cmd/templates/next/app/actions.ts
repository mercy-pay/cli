'use server';

import { MercyPay } from '@mercy-pay/next';

const mercyPay = new MercyPay(process.env.MERCY_PAY_API_KEY!);

// La session se crée côté serveur : la clé API ne va jamais dans le navigateur.
export async function startPayment(): Promise<string> {
  const session = await mercyPay.checkoutSessions.create(
    { amount: 1000, currency: 'XAF', description: 'Commande de test', ui_mode: 'embedded' },
    { idempotencyKey: `test-${Date.now()}` },
  );

  return session.client_secret!;
}
