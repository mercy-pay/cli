<?php

namespace App\Http\Controllers;

use Illuminate\View\View;
use MercyPay\Laravel\Facades\MercyPay;

class CheckoutController extends Controller
{
    /** Une commande de test payée avec le paiement intégré. */
    public function show(): View
    {
        $session = MercyPay::checkoutSessions()->create([
            'amount' => 1000,
            'currency' => 'XAF',
            'description' => 'Commande de test',
            'ui_mode' => 'embedded',
        ], ['idempotency_key' => 'test-'.str()->random(16)]);

        return view('checkout', ['clientSecret' => $session->client_secret]);
    }
}
