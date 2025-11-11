# 📄 Spécification — Validation Factur-X

**Version** : v1.3.0  
**Date** : Janvier 2025  
**Sprint** : Sprint 5 Phase 5.3  
**Statut** : ✅ Implémenté

---

## 🎯 Vue d'Ensemble

Dorevia Vault valide les factures **Factur-X** selon la norme **EN 16931** (UBL 2.1), avec extraction automatique des métadonnées et validation de la cohérence des montants.

### Fonctionnalités

- ✅ **Parsing XML Factur-X** : Extraction depuis PDF ou XML pur
- ✅ **Validation structure** : Vérification champs obligatoires EN 16931
- ✅ **Extraction métadonnées** : Numéro, dates, montants, TVA, lignes
- ✅ **Validation montants** : Cohérence TotalTTC = TotalHT + TaxAmount
- ✅ **Intégration automatique** : Utilisation dans `/api/v1/invoices`

---

## 📋 Format Factur-X

### Structure XML (UBL 2.1)

Factur-X utilise le format **UBL (Universal Business Language)** selon EN 16931 :

```xml
<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
  <ID>F2025-00123</ID>
  <IssueDate>2025-01-15</IssueDate>
  <DocumentCurrencyCode>EUR</DocumentCurrencyCode>
  <AccountingSupplierParty>
    <Party>
      <PartyName>
        <Name>ACME Corp</Name>
      </PartyName>
      <PartyTaxScheme>
        <CompanyID>FR12345678901</CompanyID>
      </PartyTaxScheme>
    </Party>
  </AccountingSupplierParty>
  <LegalMonetaryTotal>
    <TaxExclusiveAmount>158.33</TaxExclusiveAmount>
    <TaxInclusiveAmount>190.00</TaxInclusiveAmount>
  </LegalMonetaryTotal>
  <InvoiceLine>
    <!-- Lignes de facture -->
  </InvoiceLine>
</Invoice>
```

### Format PDF/A-3 avec XML Embarqué

Les factures Factur-X sont généralement des **PDF/A-3** avec le XML embarqué dans le fichier.

---

## ✅ Validation

### Champs Obligatoires (EN 16931)

| Champ | Description | Validation |
|:------|:-----------|:----------|
| `Invoice/ID` | Numéro de facture | Non vide |
| `Invoice/IssueDate` | Date d'émission | Format ISO 8601 |
| `Invoice/DocumentCurrencyCode` | Code devise | ISO 4217 (EUR, USD, etc.) |
| `AccountingSupplierParty/Party/PartyTaxScheme/CompanyID` | Numéro TVA vendeur | Non vide |

### Validation des Montants

La validation vérifie la cohérence :

```
TotalTTC = TotalHT + TaxAmount (± tolérance 0.01)
```

**Tolérance** : 1 centime pour les arrondis.

### Résultat de Validation

```go
type ValidationResult struct {
    Valid    bool     `json:"valid"`
    Errors   []string `json:"errors,omitempty"`
    Warnings []string `json:"warnings,omitempty"`
    Metadata *InvoiceMetadata `json:"metadata,omitempty"`
}
```

---

## 📊 Extraction Métadonnées

### Structure InvoiceMetadata

```go
type InvoiceMetadata struct {
    InvoiceNumber string    `json:"invoice_number"`
    InvoiceDate   time.Time `json:"invoice_date"`
    DueDate       *time.Time `json:"due_date,omitempty"`
    TotalHT       float64   `json:"total_ht"`
    TotalTTC      float64   `json:"total_ttc"`
    Currency      string    `json:"currency"`
    TaxAmount     float64   `json:"tax_amount"`
    SellerVAT     string    `json:"seller_vat"`
    BuyerVAT      string    `json:"buyer_vat"`
    SellerName    string    `json:"seller_name"`
    BuyerName     string    `json:"buyer_name"`
    LineItems     []LineItem `json:"line_items"`
}
```

### Lignes de Facture

```go
type LineItem struct {
    Description string  `json:"description"`
    Quantity    float64 `json:"quantity"`
    UnitPrice   float64 `json:"unit_price"`
    TaxRate     float64 `json:"tax_rate"`
    TotalHT     float64 `json:"total_ht"`
    TotalTTC    float64 `json:"total_ttc"`
}
```

---

## 🔧 Intégration

### Endpoint `/api/v1/invoices`

La validation Factur-X est automatiquement exécutée lors de l'ingestion :

```json
POST /api/v1/invoices
{
  "source": "sales",
  "model": "account.move",
  "file": "<base64 PDF Factur-X>",
  "meta": {
    "content_type": "application/pdf"
  }
}
```

**Comportement** :
1. Décodage du fichier base64
2. Validation Factur-X (si `FACTURX_VALIDATION_ENABLED=true`)
3. Extraction métadonnées
4. Utilisation métadonnées pour enrichir le document
5. Retour erreur si validation requise et échoue

### Configuration

```bash
# Activation validation
FACTURX_VALIDATION_ENABLED=true

# Validation requise (rejette si invalide)
FACTURX_VALIDATION_REQUIRED=false
```

### Priorité Métadonnées

1. **Métadonnées Factur-X** (si validation réussie)
2. **Métadonnées payload** (fallback)

---

## 🧪 Tests

### Tests Unitaires

- ✅ `TestFacturXValidator_Validate_ValidXML` : Validation XML valide
- ✅ `TestFacturXValidator_Validate_InvalidXML` : Validation XML invalide
- ✅ `TestFacturXValidator_Validate_ExtractMetadata` : Extraction métadonnées
- ✅ `TestFacturXValidator_Validate_AmountMismatch` : Validation montants
- ✅ `TestFacturXValidator_Validate_PDFWithXML` : Extraction depuis PDF

**Total** : 10 tests unitaires

---

## 📋 Exemples

### Facture Valide

```json
{
  "valid": true,
  "metadata": {
    "invoice_number": "F2025-00123",
    "invoice_date": "2025-01-15T00:00:00Z",
    "total_ht": 158.33,
    "total_ttc": 190.00,
    "tax_amount": 31.67,
    "currency": "EUR",
    "seller_vat": "FR12345678901",
    "buyer_vat": "FR98765432109"
  }
}
```

### Facture Invalide

```json
{
  "valid": false,
  "errors": [
    "invoice number is required",
    "invoice date is required",
    "seller VAT number is required"
  ]
}
```

---

## 🔍 Détails Techniques

### Extraction XML depuis PDF

L'algorithme recherche le marqueur `<?xml` dans le PDF et extrait le contenu jusqu'à `</Invoice>`.

**Limitation** : Nécessite que le XML soit contigu dans le PDF.

### Parsing UBL

Le parser utilise `encoding/xml` de Go avec une structure correspondant au schéma UBL 2.1.

**Namespace** : `urn:oasis:names:specification:ubl:schema:xsd:Invoice-2`

---

## 📚 Références

- [EN 16931 - European Standard for Electronic Invoicing](https://www.en16931.org/)
- [UBL 2.1 Specification](https://www.oasis-open.org/standards#ublv2.1)
- [Factur-X Documentation](https://www.factur-x.eu/)

