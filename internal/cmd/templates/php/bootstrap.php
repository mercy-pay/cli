<?php

require __DIR__.'/vendor/autoload.php';

// Les secrets viennent de .env, jamais du code.
foreach (file(__DIR__.'/.env', FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES) ?: [] as $line) {
    [$name, $value] = array_pad(explode('=', $line, 2), 2, '');
    putenv(trim($name).'='.trim($value));
}

$mercyPay = new MercyPay\MercyPayClient((string) getenv('MERCY_PAY_API_KEY'));
