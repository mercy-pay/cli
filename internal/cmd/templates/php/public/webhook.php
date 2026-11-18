<?php

require __DIR__.'/../bootstrap.php';

use MercyPay\Exception\SignatureVerificationException;
use MercyPay\Webhook;

try {
    $event = Webhook::constructEvent(
        (string) file_get_contents('php://input'),
        $_SERVER['HTTP_MERCYPAY_SIGNATURE'] ?? null,
        (string) getenv('MERCY_PAY_WEBHOOK_SECRET'),
    );
} catch (SignatureVerificationException) {
    http_response_code(400);
    exit;
}

error_log("Webhook reçu : {$event->type}");
http_response_code(200);
