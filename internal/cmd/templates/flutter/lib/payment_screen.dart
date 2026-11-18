import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;
import 'package:mercy_pay_flutter/mercy_pay_flutter.dart';

/// Un écran de paiement : votre serveur crée la session, l'app affiche le formulaire.
/// Remplacez serverUrl par l'adresse de votre serveur (jamais de clé API dans l'app).
class PaymentScreen extends StatefulWidget {
  const PaymentScreen({super.key, this.serverUrl = 'http://10.0.2.2:4242'});

  final String serverUrl;

  @override
  State<PaymentScreen> createState() => _PaymentScreenState();
}

class _PaymentScreenState extends State<PaymentScreen> {
  late final Future<String> _clientSecret = _createSession();

  Future<String> _createSession() async {
    final response = await http.post(Uri.parse('${widget.serverUrl}/session'));
    return (jsonDecode(response.body) as Map<String, dynamic>)['client_secret'] as String;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Paiement')),
      body: FutureBuilder<String>(
        future: _clientSecret,
        builder: (context, snapshot) {
          if (!snapshot.hasData) {
            return const Center(child: CircularProgressIndicator());
          }
          return MercyPayEmbeddedCheckout(
            clientSecret: snapshot.data!,
            onSuccess: (event) => Navigator.of(context).pop(true),
          );
        },
      ),
    );
  }
}
