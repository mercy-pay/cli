<!doctype html>
<html lang="fr">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Boutique de test</title>
  <script src="https://cdn.jsdelivr.net/npm/@mercy-pay/js@0.1/dist/mercy-pay.js"></script>
</head>
<body style="font-family: system-ui, sans-serif; max-width: 480px; margin: 40px auto; padding: 0 16px">
  <h1>Commande de test</h1>
  <p>1 000 FCFA</p>
  <div id="paiement"></div>
  <script>
    MercyPay.embedCheckout({
      clientSecret: @json($clientSecret),
      onSuccess: () => document.querySelector('h1').textContent = 'Merci, paiement reçu !',
    }).mount('#paiement');
  </script>
</body>
</html>
