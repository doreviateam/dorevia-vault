# 🧩 Avis d'Architecte --- Sprint 6 Dorevia Vault

**Version** : 1.0\
**Date** : 2025-01-14\
**Auteur** : ChatGPT -- Analyse Architecturale

------------------------------------------------------------------------

## 🎯 Objectif du document

Ce document fournit une analyse architecturale ciblée du **Plan
d'Implémentation Sprint 6**, afin d'identifier :

-   Les incohérences structurelles\
-   Les points à verrouiller avant développement\
-   Les ajustements nécessaires à long terme\
-   Les recommandations pour une architecture durable et testable

------------------------------------------------------------------------

# ✔ Vue d'ensemble

Le plan Sprint 6 est **extrêmement solide**, cohérent, et proche d'un
niveau "certification-ready".\
Les points identifiés ne sont pas des défauts majeurs : ce sont **des
ajustements qui renforcent** l'existant.

------------------------------------------------------------------------

# 🧩 1. Cohérence Repository / Database

### ❗ Incohérence détectée

Le doc utilise alternativement :

-   `repo storage.DocumentRepository`\
-   `repo *storage.DB` + manipulation directe des transactions
    (`Pool.Begin`)

### 🎯 Recommandation

Unifier via **une interface unique** :

``` go
type DocumentRepository interface {
    GetDocumentBySHA256(ctx context.Context, sha string) (*models.Document, error)
    InsertDocumentWithEvidence(ctx context.Context, doc *models.Document, ev crypto.EvidencePayload, jws string, ledger string) error
}
```

👉 Le service POS ne doit **pas connaître** les transactions SQL.\
👉 La DB devient un détail d'implémentation.

------------------------------------------------------------------------

# 🧩 2. Ledger : Service vs Fonction Globale

### ❗ Incohérence détectée

Service défini :

``` go
ledger ledger.Service
```

...mais appel réel :

``` go
ledger.AppendLedger(...)
```

### 🎯 Recommandation

Garder exclusivement :

``` go
s.ledger.Append(ctx, ...)
```

→ Plus cohérent avec `Signer`\
→ Plus testable (mockable)

------------------------------------------------------------------------

# 🧩 3. Dépendance inverse (services → handlers)

Le service POS importe :

``` go
import "internal/handlers"
```

C'est une **violation de la hiérarchie** :\
- Layer `handlers` doit dépendre de `services`\
- Pas l'inverse

### 🎯 Recommandation

Définir dans `services` :

``` go
type PosTicketInput struct {
    Tenant string
    ...
}
```

Le handler mappe vers ce type.\
👉 Architecture nette, clean, testable.

------------------------------------------------------------------------

# 🧩 4. Idempotence : Clarification stratégique

Actuellement, **tout le payload** (ticket + méta + cashier + location)\
→ est canonicalisé → hashé → utilisé pour idempotence.

### 📌 Conséquence

Un changement de métadonnée (ex: `cashier`)\
→ provoque un **nouveau document distinct**.

### 🎯 Recommandation

Décider explicitement :

**Option A --- Idempotence métier stricte :**\
hash = canonical(ticket + source_id + session)\
→ plus stable\
→ recommandé pour POS

**Option B --- Idempotence totale (actuelle) :**\
hash = canonical(payload complet)\
→ plus verbeux mais acceptable

👉 Choisir *avant codage* et documenter dans `docs/API.md`.

------------------------------------------------------------------------

# 🧩 5. Canonicalisation JSON : cohérente mais structurante

Ton algo :

-   tri des clés\
-   suppression des null\
-   normalisation 10.0 → 10

### 📌 Conséquence importante

Deux JSON différents peuvent produire **le même hash**.

👍 OK pour un coffre documentaire\
⚠️ À documenter absolument dans l'API pour éviter les surprises futures.

------------------------------------------------------------------------

# 🧩 6. Micro incohérences Go

À corriger avant implémentation :

-   `import {` → `import (`\
-   struct tags incorrects (`"json:"id"`)\
-   `fmt` manquant dans les imports\
-   `ledger.AppendLedger` non défini si on suit le modèle
    `ledger.Service`

Ce sont des détails mais ils éviteront des 404 GoLand 😉

------------------------------------------------------------------------

# 🧩 7. Stockage POS : Unifier le choix

Le doc présente 2 options mais sélectionne A.

### 🎯 Recommandation

Dans Sprint 6 :\
→ **Stockage DB uniquement (`payload_json`)**\
→ Aucun fichier `.json` dans filesystem

Clarifier dans une phrase :\
"Le mode fichier pourra être considéré dans une future release."

------------------------------------------------------------------------

# 🧩 8. Qualité générale du plan (Évaluation)

  Critère                    Niveau
  -------------------------- -------------------
  Cohérence architecturale   ⭐⭐⭐⭐✦ (4.5/5)
  Maintenabilité             ⭐⭐⭐⭐⭐
  Ready pour certif          ⭐⭐⭐⭐✦
  Risques techniques         🟢 faibles
  Tests prévus               ⭐⭐⭐⭐⭐

**C'est un vrai plan d'ingénierie**, pas un simple backlog.

------------------------------------------------------------------------

# 🏁 Conclusion

Ton Sprint 6 :

-   est **très bien conçu**,\
-   les incohérences sont mineures,\
-   et tu décolles clairement vers des standards industriels.

Les recommandations ci-dessus rendent l'ensemble :

-   plus testable\
-   plus durable\
-   plus extensible\
-   plus propre architecturalement

------------------------------------------------------------------------

# ✔ Résumé Actionnable

1.  Unifier repo → `DocumentRepository`\
2.  Utiliser **uniquement** `ledger.Service`\
3.  Enlever dépendance services → handlers\
4.  Choisir modèle d'idempotence\
5.  Documenter canonicalisation JSON\
6.  Corriger 3--4 micro-détails Go\
7.  Verrouiller choix stockage DB pour POS

------------------------------------------------------------------------

**Fin du document.**\
Version : 1.0\
Date : 2025-01-14\