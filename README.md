# mercypay

La ligne de commande de [Mercy Pay](https://mercy-pay.com) : connectez-vous, recevez vos webhooks sur votre ordinateur, déclenchez des événements de test et appelez l'API.

## Installer

macOS et Linux, avec Homebrew :

```sh
brew install mercy-pay/tap/mercypay
```

Ou avec le script d'installation :

```sh
curl -fsSL https://raw.githubusercontent.com/mercy-pay/cli/main/install.sh | sh
```

Windows : téléchargez l'archive `.zip` de la [dernière version](https://github.com/mercy-pay/cli/releases/latest).

## Se connecter

```sh
mercypay login
```

Votre navigateur s'ouvre. Vérifiez le code, choisissez la boutique et autorisez la CLI : elle reçoit une clé de **sandbox**, rangée dans le trousseau de votre système. Elle se révoque dans Développeurs, onglet Clés API.

Pour utiliser une autre clé, live comprise : `mercypay login --api-key mp_live_…`, ou `--api-key` sur n'importe quelle commande, ou la variable `MERCYPAY_API_KEY`.

## Recevoir vos webhooks en local

```sh
mercypay listen --forward-to localhost:8000/webhooks/mercy-pay
```

Chaque nouvel événement du compte est envoyé à votre serveur, signé comme un vrai webhook (`MercyPay-Signature`). Le secret de signature s'affiche au démarrage et reste le même d'une fois à l'autre : mettez-le dans la configuration de votre app locale. `mercypay listen --print-secret` l'affiche seul.

Les événements sont lus toutes les 2 secondes. `--events payment.succeeded,payment.failed` ne relaie que ces types.

## Démarrer un projet

```sh
mercypay init node        # ou next, php, laravel, django, flutter
```

Un projet d'exemple prêt à lancer : une session de paiement intégrée, la route des webhooks, et votre clé sandbox déjà dans le fichier d'environnement (jamais une clé live). Le `README.md` du projet dit la suite.

## Faire un vrai paiement de test

```sh
mercypay payments create --test succeeded --provider MTN_MOMO_CMR --wait
mercypay payments create --test insufficient_balance --provider ORANGE_CMR --wait
```

En sandbox, la CLI prend le numéro de test qui produit ce résultat, lance le paiement et attend son statut final. Les résultats : `succeeded`, `pending`, `failed`, ou un `failure_code` comme `insufficient_balance`. `--amount` et `--currency` changent le montant (1 000 XAF par défaut). `mercypay test-helpers mobile-money-numbers -o table` liste tous les numéros.

## Déclencher un événement de test

```sh
mercypay trigger payment.succeeded
mercypay trigger --list
```

En sandbox seulement. L'événement porte des données d'exemple au format de l'objet réel et part vers vos endpoints sandbox et vers `mercypay listen`. Rien d'autre n'est créé.

## Suivre vos appels à l'API

```sh
mercypay logs tail
mercypay logs tail --status error --path payments
```

## Appeler l'API

```sh
mercypay payments list -d limit=5
mercypay payments get pay_5e0c8a7b3d1f4a2e9c648b7f0d2e1a93
mercypay customers create -d name="Awa Ndiaye" -d email=awa@example.com
mercypay checkout-sessions create -d amount:=15000 -d currency=XAF -d metadata[order_id]=4821
mercypay get /v1/account
```

- `-d cle=valeur` envoie du texte, `-d cle:=json` un nombre, un booléen ou un objet JSON.
- `metadata[cle]=valeur` remplit un objet, `liste[]=valeur` une liste.
- Un `POST` porte toujours une `Idempotency-Key` : une nouvelle tentative après une coupure ne crée rien en double. `-i` choisit la clé.

- `--wait` attend le statut final d'un paiement ou d'une session : `mercypay payments get pay_… --wait`.
- `mercypay checkout-sessions create -d amount:=15000 -d currency=XAF --open` ouvre la page de paiement dans le navigateur.
- `-o table` affiche une liste en colonnes, `-o compact` une seule ligne de JSON (pour `jq`). Le JSON indenté reste le format par défaut.

`mercypay open dashboard`, `mercypay open docs` ou `mercypay open payments` ouvre une page dans le navigateur.

`mercypay <commande> --help` détaille chaque commande. La référence complète de l'API est sur [docs.mercy-pay.com](https://docs.mercy-pay.com).

## Profils

`--profile boutique-2` garde une connexion séparée par boutique. `mercypay config list` montre les profils, `mercypay config use boutique-2` choisit celui utilisé par défaut. La configuration est dans `~/.config/mercypay/config.json`.

## Autocomplétion

Installée avec Homebrew. Sinon : `mercypay completion zsh > "${fpath[1]}/_mercypay"` (aussi `bash` et `fish`). Elle propose les commandes, les opérateurs, les types d'événements et les stacks de `init`.

## Développement

```sh
go test ./...
go build -o mercypay .
MERCYPAY_API_BASE=http://localhost:8000 ./mercypay login
```

`testdata/openapi.json` est une copie de la référence de l'API : un test vérifie que chaque opération a sa commande. Mettez-la à jour quand l'API change.
