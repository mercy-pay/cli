import { createWebhookHandler } from '@mercy-pay/next';

export const POST = createWebhookHandler({
  handlers: {
    'payment.succeeded': async (event) => {
      console.log('Paiement réussi :', event.data.id);
    },
  },
});
