'use client';

import { EmbeddedCheckout } from '@mercy-pay/react';
import { useState } from 'react';

export function Payment({ clientSecret }: { clientSecret: string }) {
  const [paid, setPaid] = useState(false);

  if (paid) {
    return <p>Merci, paiement reçu !</p>;
  }

  return <EmbeddedCheckout clientSecret={clientSecret} onSuccess={() => setPaid(true)} />;
}
