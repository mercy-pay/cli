# Mercy Pay dans votre app Flutter

1. `flutter pub add mercy_pay_flutter http`
2. Copiez `lib/payment_screen.dart` dans votre app et ouvrez `PaymentScreen`.
3. L'app a besoin d'un serveur qui crée la session : lancez par exemple `mercypay init node` dans un autre dossier, puis `npm start`. Sur l'émulateur Android, ce serveur répond à `http://10.0.2.2:4242`.

La clé API reste sur le serveur, jamais dans l'app : une app se décompile.

Guide : https://docs.mercy-pay.com/payments/embedded#dans-une-app-mobile
