# Journal des changements

## 0.2.0 (18 novembre 2026)

- Nouveau : `mercypay init <stack>` crée un projet d'exemple prêt à lancer (Node, Next.js, PHP, Laravel, Django, Flutter).
- Nouveau : `payments create --test <résultat> --provider <opérateur>` lance un vrai paiement sandbox avec le bon numéro de test.
- Nouveau : `--wait` sur `payments create|get` et `checkout-sessions create|get`, et `--open` sur `checkout-sessions create`.
- Nouveau : `-o json|compact|table`, `mercypay open`, `mercypay config list|use`.
- Nouveau : `embed-domains` et `test-helpers mobile-money-numbers`.
- L'autocomplétion (bash, zsh, fish) s'installe avec Homebrew et propose les opérateurs et les types d'événements.

## 0.1.0 (16 novembre 2026)

- Première version : `login`, `listen`, `trigger`, `logs` et l'appel de chaque endpoint de l'API.
