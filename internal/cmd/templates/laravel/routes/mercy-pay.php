<?php

// À ajouter dans routes/web.php
use App\Http\Controllers\CheckoutController;
use Illuminate\Support\Facades\Route;

Route::get('/paiement-test', [CheckoutController::class, 'show']);
