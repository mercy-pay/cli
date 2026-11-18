import { startPayment } from './actions';
import { Payment } from './payment';

export const dynamic = 'force-dynamic';

export default async function Page() {
  const clientSecret = await startPayment();

  return (
    <main>
      <h1>Commande de test</h1>
      <p>1 000 FCFA</p>
      <Payment clientSecret={clientSecret} />
    </main>
  );
}
