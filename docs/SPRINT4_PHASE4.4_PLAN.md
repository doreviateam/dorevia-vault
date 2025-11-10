# 📋 Plan Détaillé — Sprint 4 Phase 4.4 : Audit & Conformité

**Version** : v1.2-dev  
**Date de démarrage** : Février 2025  
**Responsable** : Doreviateam (David Baron)  
**Statut** : 🟡 Planification — Prêt pour exécution  
**Durée prévue** : **4 jours** (J1-J4)

---

## 🎯 Objectif Global

Générer des **rapports d'audit consolidés** (mensuels/trimestriels) pour la conformité réglementaire (PDP/PPF 2026), avec export multi-format (JSON, CSV, PDF) et signature JWS.

---

## 📊 Vue d'Ensemble

### Contexte

La Phase 4.4 complète le Sprint 4 en permettant la génération de rapports d'audit consolidés à partir de :
- **Logs d'audit** (Phase 4.2) : Événements journaliers signés
- **Métriques Prometheus** (Phase 4.1) : Statistiques système et métier
- **Base de données** : Documents, ledger, réconciliations
- **Signatures journalières** : Preuves d'intégrité des logs

### Objectifs Concrets

1. **Consolider** → Agréger données logs + métriques + DB sur une période
2. **Analyser** → Calculer statistiques (documents, erreurs, performances, ledger)
3. **Exporter** → Générer rapports JSON, CSV, PDF
4. **Signer** → Ajouter signature JWS pour preuve d'intégrité
5. **Automatiser** → CLI pour génération manuelle ou scriptable

---

## 🏗️ Architecture Technique

### Structure des Modules

```
internal/audit/
├── log.go          ✅ Existant (Phase 4.2)
├── sign.go         ✅ Existant (Phase 4.2)
├── export.go       ✅ Existant (Phase 4.2)
├── rotation.go     ✅ Existant (Phase 4.2)
├── odoo_export.go  ✅ Existant (Phase 4.3)
├── report.go       ⏳ NOUVEAU (Phase 4.4)
└── pdf.go          ⏳ NOUVEAU (Phase 4.4)

cmd/audit/
└── main.go         ⏳ NOUVEAU (Phase 4.4)
```

### Dépendances Externes

| Module | Version | Usage |
|:-------|:--------|:------|
| `github.com/jung-kurt/gofpdf` | v2.4.0+ | Génération PDF |
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | Signature JWS (déjà présent) |
| `github.com/prometheus/client_golang` | v1.23.2 | Lecture métriques (déjà présent) |

---

## 📑 Structure du Rapport

### Type Go `AuditReport`

```go
package audit

import "time"

// PeriodType représente le type de période
type PeriodType string

const (
	PeriodTypeMonthly   PeriodType = "monthly"
	PeriodTypeQuarterly PeriodType = "quarterly"
	PeriodTypeCustom    PeriodType = "custom"
)

// Period représente une période de rapport
type Period struct {
	Type      PeriodType `json:"type"`       // monthly, quarterly, custom
	StartDate string     `json:"start_date"`  // YYYY-MM-DD
	EndDate   string     `json:"end_date"`    // YYYY-MM-DD
	Label     string     `json:"label"`       // "Janvier 2025", "Q1 2025", etc.
}

// ReportSummary représente le résumé exécutif
type ReportSummary struct {
	TotalDocuments      int64   `json:"total_documents"`       // Total documents vaultés
	TotalErrors         int64   `json:"total_errors"`          // Total erreurs
	ErrorRate           float64 `json:"error_rate"`            // Taux d'erreur (%)
	TotalLedgerEntries  int64   `json:"total_ledger_entries"`  // Total entrées ledger
	TotalReconciliations int64  `json:"total_reconciliations"` // Total réconciliations
	AvgDocumentSize     int64   `json:"avg_document_size"`     // Taille moyenne document (bytes)
	TotalStorageSize    int64   `json:"total_storage_size"`     // Taille totale stockage (bytes)
}

// DocumentStats représente les statistiques sur les documents
type DocumentStats struct {
	Total           int64             `json:"total"`            // Total documents
	ByStatus        map[string]int64   `json:"by_status"`      // Par statut (success, error, idempotent)
	BySource        map[string]int64   `json:"by_source"`       // Par source (sales, purchase, pos, etc.)
	ByContentType   map[string]int64   `json:"by_content_type"` // Par type MIME
	SizeDistribution SizeDistribution `json:"size_distribution"` // Distribution des tailles
}

// SizeDistribution représente la distribution des tailles de documents
type SizeDistribution struct {
	Min    int64   `json:"min"`     // Taille minimale (bytes)
	Max    int64   `json:"max"`     // Taille maximale (bytes)
	Mean   float64 `json:"mean"`    // Taille moyenne (bytes)
	Median int64  `json:"median"`    // Taille médiane (bytes)
	P95    int64   `json:"p95"`     // Percentile 95 (bytes)
	P99    int64   `json:"p99"`     // Percentile 99 (bytes)
}

// ErrorStats représente les statistiques sur les erreurs
type ErrorStats struct {
	Total           int64             `json:"total"`            // Total erreurs
	ByType          map[string]int64  `json:"by_type"`         // Par type d'erreur
	ByEventType     map[string]int64  `json:"by_event_type"`   // Par type d'événement
	CriticalErrors  []CriticalError   `json:"critical_errors"` // Erreurs critiques (top 10)
}

// CriticalError représente une erreur critique
type CriticalError struct {
	Timestamp  string `json:"timestamp"`   // RFC3339
	EventType  string `json:"event_type"`  // Type d'événement
	DocumentID string `json:"document_id"`  // ID document (si applicable)
	Message    string `json:"message"`      // Message d'erreur
	Count      int64  `json:"count"`       // Nombre d'occurrences
}

// PerformanceStats représente les statistiques de performance
type PerformanceStats struct {
	DocumentStorage PerformanceMetric `json:"document_storage"` // Stockage documents
	JWSSignature    PerformanceMetric `json:"jws_signature"`   // Signature JWS
	LedgerAppend    PerformanceMetric `json:"ledger_append"`    // Ajout ledger
	Transaction     PerformanceMetric `json:"transaction"`     // Transactions
}

// PerformanceMetric représente une métrique de performance
type PerformanceMetric struct {
	Count   int64   `json:"count"`    // Nombre d'observations
	Mean    float64 `json:"mean"`     // Durée moyenne (secondes)
	Median  float64 `json:"median"`   // Durée médiane (secondes)
	P50     float64 `json:"p50"`     // Percentile 50 (secondes)
	P95     float64 `json:"p95"`     // Percentile 95 (secondes)
	P99     float64 `json:"p99"`     // Percentile 99 (secondes)
	Min     float64 `json:"min"`     // Durée minimale (secondes)
	Max     float64 `json:"max"`     // Durée maximale (secondes)
}

// LedgerStats représente les statistiques sur le ledger
type LedgerStats struct {
	TotalEntries     int64   `json:"total_entries"`      // Total entrées
	NewEntries       int64   `json:"new_entries"`        // Nouvelles entrées (période)
	Errors           int64   `json:"errors"`              // Erreurs ledger
	ErrorRate        float64 `json:"error_rate"`         // Taux d'erreur (%)
	CurrentSize      int64   `json:"current_size"`        // Taille actuelle
	ChainIntegrity   bool    `json:"chain_integrity"`    // Intégrité chaîne (vérifiée)
	LastHash         string  `json:"last_hash"`           // Dernier hash
}

// ReconciliationStats représente les statistiques sur les réconciliations
type ReconciliationStats struct {
	TotalRuns        int64   `json:"total_runs"`         // Total exécutions
	SuccessfulRuns  int64   `json:"successful_runs"`    // Exécutions réussies
	FailedRuns       int64   `json:"failed_runs"`        // Exécutions échouées
	OrphanFilesFound int64  `json:"orphan_files_found"` // Fichiers orphelins trouvés
	OrphanFilesFixed int64  `json:"orphan_files_fixed"` // Fichiers orphelins corrigés
	DocumentsFixed   int64   `json:"documents_fixed"`   // Documents corrigés
}

// DailySignature représente une signature journalière
type DailySignature struct {
	Date      string `json:"date"`       // YYYY-MM-DD
	Hash      string `json:"hash"`       // SHA256 hash
	JWS       string `json:"jws"`       // Signature JWS
	LineCount int64  `json:"line_count"` // Nombre de lignes
	Timestamp string `json:"timestamp"` // RFC3339
}

// ReportMetadata représente les métadonnées du rapport
type ReportMetadata struct {
	GeneratedAt    string `json:"generated_at"`     // RFC3339
	GeneratedBy    string `json:"generated_by"`     // "dorevia-vault" ou "cli"
	Version        string `json:"version"`          // Version du système
	ReportID       string `json:"report_id"`        // UUID unique du rapport
	ReportHash     string `json:"report_hash"`       // SHA256 du rapport JSON
	ReportJWS      string `json:"report_jws"`       // Signature JWS du rapport (si signé)
	DataSources    []string `json:"data_sources"`   // Sources de données utilisées
}

// AuditReport représente le rapport d'audit complet
type AuditReport struct {
	Period         Period              `json:"period"`
	Summary        ReportSummary       `json:"summary"`
	Documents      DocumentStats       `json:"documents"`
	Errors         ErrorStats          `json:"errors"`
	Performance    PerformanceStats    `json:"performance"`
	Ledger         LedgerStats        `json:"ledger"`
	Reconciliation ReconciliationStats `json:"reconciliation"`
	Signatures     []DailySignature    `json:"signatures"` // Signatures journalières de la période
	Metadata       ReportMetadata      `json:"metadata"`
}
```

---

## 🔧 Implémentation Détaillée

### J1 : Module `internal/audit/report.go` (Génération JSON/CSV)

#### Objectifs

1. Créer le module `report.go` avec génération de rapports
2. Implémenter la collecte de données depuis :
   - Logs d'audit (via `Exporter`)
   - Base de données PostgreSQL
   - Métriques Prometheus (optionnel, via scraping)
3. Calculer toutes les statistiques
4. Générer export JSON et CSV

#### Fonctions Principales

```go
// ReportGenerator génère des rapports d'audit
type ReportGenerator struct {
	logger      *Logger
	exporter    *Exporter
	db          *storage.DB  // Optionnel (si DB disponible)
	jwsService  *crypto.Service  // Pour signature rapport
	log         zerolog.Logger
}

// NewReportGenerator crée un nouveau générateur de rapports
func NewReportGenerator(logger *Logger, exporter *Exporter, db *storage.DB, jwsService *crypto.Service, log zerolog.Logger) *ReportGenerator

// Generate génère un rapport pour une période donnée
func (g *ReportGenerator) Generate(periodType PeriodType, startDate, endDate string) (*AuditReport, error)

// GenerateMonthly génère un rapport mensuel
func (g *ReportGenerator) GenerateMonthly(year int, month int) (*AuditReport, error)

// GenerateQuarterly génère un rapport trimestriel
func (g *ReportGenerator) GenerateQuarterly(year int, quarter int) (*AuditReport, error)

// collectAuditEvents collecte les événements d'audit depuis les logs
func (g *ReportGenerator) collectAuditEvents(startDate, endDate string) ([]Event, error)

// collectDocumentStats collecte les statistiques documents depuis la DB
func (g *ReportGenerator) collectDocumentStats(startDate, endDate string) (*DocumentStats, error)

// collectErrorStats collecte les statistiques d'erreurs depuis les logs
func (g *ReportGenerator) collectErrorStats(events []Event) (*ErrorStats, error)

// collectPerformanceStats collecte les statistiques de performance depuis les logs
func (g *ReportGenerator) collectPerformanceStats(events []Event) (*PerformanceStats, error)

// collectLedgerStats collecte les statistiques ledger depuis la DB
func (g *ReportGenerator) collectLedgerStats(startDate, endDate string) (*LedgerStats, error)

// collectReconciliationStats collecte les statistiques réconciliation depuis les logs
func (g *ReportGenerator) collectReconciliationStats(events []Event) (*ReconciliationStats, error)

// collectDailySignatures collecte les signatures journalières
func (g *ReportGenerator) collectDailySignatures(startDate, endDate string) ([]DailySignature, error)

// calculateSummary calcule le résumé exécutif
func (g *ReportGenerator) calculateSummary(docs *DocumentStats, errors *ErrorStats, ledger *LedgerStats, recon *ReconciliationStats) *ReportSummary

// Sign signe le rapport avec JWS
func (g *ReportGenerator) Sign(report *AuditReport) error

// ExportJSON exporte le rapport en JSON
func (g *ReportGenerator) ExportJSON(report *AuditReport, outputPath string) error

// ExportCSV exporte le rapport en CSV (format simplifié)
func (g *ReportGenerator) ExportCSV(report *AuditReport, outputPath string) error
```

#### Collecte de Données

**1. Logs d'audit** (via `Exporter` existant) :
```go
// Utiliser Exporter.Export() pour récupérer tous les événements de la période
opts := ExportOptions{
    From:   startDate,
    To:     endDate,
    Page:   1,
    Limit:  10000, // Max pour récupérer tout
    Format: ExportFormatJSON,
}
result, err := g.exporter.Export(opts)
```

**2. Base de données** (si `DATABASE_URL` configuré) :
```sql
-- Statistiques documents
SELECT 
    COUNT(*) as total,
    status,
    source,
    content_type,
    AVG(size_bytes) as avg_size,
    PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY size_bytes) as median,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY size_bytes) as p95,
    PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY size_bytes) as p99
FROM documents
WHERE created_at >= $1 AND created_at <= $2
GROUP BY status, source, content_type;

-- Statistiques ledger
SELECT 
    COUNT(*) as total_entries,
    COUNT(CASE WHEN evidence_jws IS NULL THEN 1 END) as errors
FROM ledger
WHERE timestamp >= $1 AND timestamp <= $2;

-- Vérification intégrité chaîne
SELECT 
    l1.hash,
    l1.previous_hash,
    l2.hash as next_hash
FROM ledger l1
LEFT JOIN ledger l2 ON l2.previous_hash = l1.hash
WHERE l1.timestamp >= $1 AND l1.timestamp <= $2
ORDER BY l1.timestamp;
```

**3. Signatures journalières** (via `Signer` existant) :
```go
// Lire les signatures depuis audit/signatures/
// Format: audit-YYYY-MM-DD.log.jws
```

#### Calcul des Statistiques

**Performance (P50, P95, P99)** :
```go
// Extraire durées depuis events[].DurationMS
// Calculer percentiles avec sort + index
func calculatePercentiles(durations []float64) (p50, p95, p99 float64) {
    sort.Float64s(durations)
    n := len(durations)
    if n == 0 {
        return 0, 0, 0
    }
    p50 = durations[n*50/100]
    p95 = durations[n*95/100]
    p99 = durations[n*99/100]
    return
}
```

#### Export CSV

Format simplifié avec colonnes principales :
```csv
period_type,period_start,period_end,total_documents,total_errors,error_rate,avg_document_size,total_storage_size
monthly,2025-01-01,2025-01-31,1234,12,0.97,45678,56789012
```

**Livrables J1** :
- ✅ `internal/audit/report.go` (module complet)
- ✅ Tests unitaires `tests/unit/audit_report_test.go` (15+ tests)
- ✅ Documentation inline

---

### J2 : Module `internal/audit/pdf.go` (Génération PDF)

#### Objectifs

1. Créer le module `pdf.go` avec génération PDF
2. Implémenter template professionnel (en-tête, pied de page, graphiques)
3. Intégrer signature JWS (QR code du hash)
4. Support multi-pages

#### Dépendance

```bash
go get github.com/jung-kurt/gofpdf/v2
```

#### Structure PDF

```
Page 1 : Page de garde
├── Logo Doreviateam (optionnel)
├── Titre : "Rapport d'Audit Dorevia Vault"
├── Période : "Janvier 2025" ou "Q1 2025"
├── Date de génération
└── Signature JWS (QR code)

Page 2 : Résumé exécutif
├── Tableau récapitulatif
│   ├── Total documents
│   ├── Taux d'erreur
│   ├── Taille stockage
│   └── Intégrité ledger
└── Graphique : Évolution documents (si données disponibles)

Page 3 : Statistiques Documents
├── Tableau : Par statut (success, error, idempotent)
├── Tableau : Par source (sales, purchase, pos, etc.)
└── Graphique : Distribution par source (camembert)

Page 4 : Statistiques Erreurs
├── Tableau : Top 10 erreurs critiques
├── Graphique : Évolution erreurs (courbe)
└── Détails erreurs critiques

Page 5 : Performance
├── Tableau : Durées moyennes (P50, P95, P99)
│   ├── Stockage documents
│   ├── Signature JWS
│   ├── Ajout ledger
│   └── Transactions
└── Graphique : Latence P95 (barres)

Page 6 : Ledger & Réconciliation
├── Statistiques ledger
├── Vérification intégrité chaîne
└── Statistiques réconciliation

Page 7 : Signatures Journalières
├── Tableau : Liste signatures (date, hash, JWS)
└── Vérification intégrité

Page 8 : Métadonnées
├── Informations système
├── Sources de données
└── Signature JWS complète (texte)
```

#### Fonctions Principales

```go
// PDFGenerator génère des rapports PDF
type PDFGenerator struct {
	jwsService *crypto.Service  // Pour signature
	log        zerolog.Logger
}

// NewPDFGenerator crée un nouveau générateur PDF
func NewPDFGenerator(jwsService *crypto.Service, log zerolog.Logger) *PDFGenerator

// Generate génère un PDF à partir d'un AuditReport
func (g *PDFGenerator) Generate(report *AuditReport, outputPath string) error

// addCoverPage ajoute la page de garde
func (g *PDFGenerator) addCoverPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addSummaryPage ajoute la page résumé exécutif
func (g *PDFGenerator) addSummaryPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addDocumentStatsPage ajoute la page statistiques documents
func (g *PDFGenerator) addDocumentStatsPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addErrorStatsPage ajoute la page statistiques erreurs
func (g *PDFGenerator) addErrorStatsPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addPerformancePage ajoute la page performance
func (g *PDFGenerator) addPerformancePage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addLedgerPage ajoute la page ledger & réconciliation
func (g *PDFGenerator) addLedgerPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addSignaturesPage ajoute la page signatures journalières
func (g *PDFGenerator) addSignaturesPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addMetadataPage ajoute la page métadonnées
func (g *PDFGenerator) addMetadataPage(pdf *gofpdf.Fpdf, report *AuditReport) error

// addQRCode ajoute un QR code (hash SHA256 du rapport)
func (g *PDFGenerator) addQRCode(pdf *gofpdf.Fpdf, hash string, x, y, size float64) error

// addTable ajoute un tableau formaté
func (g *PDFGenerator) addTable(pdf *gofpdf.Fpdf, headers []string, rows [][]string) error

// addChart ajoute un graphique (barres ou camembert simple)
func (g *PDFGenerator) addChart(pdf *gofpdf.Fpdf, chartType string, data map[string]float64, x, y, w, h float64) error
```

#### Template PDF

Utiliser `gofpdf` avec :
- **Police** : Arial (ou Helvetica)
- **Taille** : 10pt (corps), 14pt (titres), 8pt (pied de page)
- **Marges** : 20mm (haut/bas), 15mm (gauche/droite)
- **Couleurs** : 
  - Bleu Dorevia (#0066CC) pour en-têtes
  - Gris (#666666) pour texte secondaire
  - Rouge (#CC0000) pour erreurs
  - Vert (#00CC00) pour succès

#### QR Code

Utiliser `github.com/skip2/go-qrcode` pour générer QR code du hash SHA256 :
```go
import "github.com/skip2/go-qrcode"

func generateQRCode(hash string) ([]byte, error) {
    return qrcode.Encode(hash, qrcode.Medium, 256)
}
```

**Livrables J2** :
- ✅ `internal/audit/pdf.go` (module complet)
- ✅ Tests unitaires `tests/unit/audit_pdf_test.go` (10+ tests)
- ✅ Exemple PDF généré

---

### J3 : CLI `cmd/audit/main.go`

#### Objectifs

1. Créer le CLI `cmd/audit/main.go`
2. Implémenter tous les flags (`--period`, `--format`, `--output`, `--sign`)
3. Gestion erreurs et validation
4. Intégration avec modules `report.go` et `pdf.go`

#### Flags CLI

```bash
./bin/audit [OPTIONS]

Options:
  --period TYPE          Type de période (monthly, quarterly, custom) [required]
  --year YEAR            Année (pour monthly/quarterly) [default: année actuelle]
  --month MONTH          Mois 1-12 (pour monthly) [default: mois actuel]
  --quarter QUARTER      Trimestre 1-4 (pour quarterly) [default: trimestre actuel]
  --from DATE            Date début YYYY-MM-DD (pour custom) [required si custom]
  --to DATE              Date fin YYYY-MM-DD (pour custom) [required si custom]
  --format FORMAT        Format d'export (json, csv, pdf) [default: json]
  --output PATH          Chemin fichier de sortie [default: stdout pour json/csv, report-YYYY-MM-DD.pdf pour pdf]
  --sign                 Signer le rapport avec JWS [default: false]
  --jws-key-path PATH    Chemin clé privée JWS [default: JWS_PRIVATE_KEY_PATH env]
  --audit-dir PATH       Répertoire audit [default: AUDIT_DIR env]
  --database-url URL      URL base de données [default: DATABASE_URL env]
  --verbose              Mode verbeux
  --help                 Afficher l'aide
```

#### Exemples d'Utilisation

```bash
# Rapport mensuel JSON (Janvier 2025)
./bin/audit --period monthly --year 2025 --month 1 --format json --output report-2025-01.json

# Rapport trimestriel PDF signé (Q1 2025)
./bin/audit --period quarterly --year 2025 --quarter 1 --format pdf --sign --output report-Q1-2025.pdf

# Rapport personnalisé CSV (15 jours)
./bin/audit --period custom --from 2025-01-15 --to 2025-01-31 --format csv --output report-custom.csv

# Rapport mensuel JSON signé (mois actuel)
./bin/audit --period monthly --format json --sign --output report-current.json
```

#### Structure CLI

```go
package main

import (
    "flag"
    "fmt"
    "os"
    "time"
    
    "github.com/doreviateam/dorevia-vault/internal/audit"
    "github.com/doreviateam/dorevia-vault/internal/config"
    "github.com/doreviateam/dorevia-vault/internal/crypto"
    "github.com/doreviateam/dorevia-vault/internal/storage"
    "github.com/doreviateam/dorevia-vault/pkg/logger"
    "github.com/rs/zerolog"
)

func main() {
    // Parse flags
    periodType := flag.String("period", "", "Type de période (monthly, quarterly, custom)")
    year := flag.Int("year", time.Now().Year(), "Année")
    month := flag.Int("month", int(time.Now().Month()), "Mois (1-12)")
    quarter := flag.Int("quarter", getCurrentQuarter(), "Trimestre (1-4)")
    fromDate := flag.String("from", "", "Date début YYYY-MM-DD (pour custom)")
    toDate := flag.String("to", "", "Date fin YYYY-MM-DD (pour custom)")
    format := flag.String("format", "json", "Format (json, csv, pdf)")
    outputPath := flag.String("output", "", "Chemin fichier de sortie")
    sign := flag.Bool("sign", false, "Signer le rapport avec JWS")
    jwsKeyPath := flag.String("jws-key-path", "", "Chemin clé privée JWS")
    auditDir := flag.String("audit-dir", "", "Répertoire audit")
    databaseURL := flag.String("database-url", "", "URL base de données")
    verbose := flag.Bool("verbose", false, "Mode verbeux")
    help := flag.Bool("help", false, "Afficher l'aide")
    
    flag.Parse()
    
    if *help {
        printHelp()
        os.Exit(0)
    }
    
    // Validation
    if err := validateFlags(periodType, fromDate, toDate); err != nil {
        fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
        os.Exit(1)
    }
    
    // Configuration
    cfg := loadConfig(auditDir, databaseURL, jwsKeyPath)
    log := logger.New(cfg.LogLevel)
    if *verbose {
        log = log.Level(zerolog.DebugLevel)
    }
    
    // Initialisation modules
    auditLogger, err := audit.NewLogger(audit.Config{
        AuditDir:      cfg.AuditDir,
        MaxBuffer:     1000,
        FlushInterval: 10 * time.Second,
        Logger:        *log,
    })
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to initialize audit logger")
    }
    defer auditLogger.Close()
    
    exporter := audit.NewExporter(auditLogger)
    
    var db *storage.DB
    if cfg.DatabaseURL != "" {
        ctx := context.Background()
        db, err = storage.NewDB(ctx, cfg.DatabaseURL, log)
        if err != nil {
            log.Warn().Err(err).Msg("Failed to connect to database, continuing without DB stats")
        } else {
            defer db.Close()
        }
    }
    
    var jwsService *crypto.Service
    if *sign || cfg.JWSPrivateKeyPath != "" {
        jwsService, err = crypto.NewService(cfg.JWSPrivateKeyPath, cfg.JWSPublicKeyPath, cfg.JWSKID)
        if err != nil {
            log.Warn().Err(err).Msg("JWS service unavailable, signature disabled")
        }
    }
    
    // Génération rapport
    generator := audit.NewReportGenerator(auditLogger, exporter, db, jwsService, *log)
    
    var report *audit.AuditReport
    var err error
    
    switch *periodType {
    case "monthly":
        report, err = generator.GenerateMonthly(*year, *month)
    case "quarterly":
        report, err = generator.GenerateQuarterly(*year, *quarter)
    case "custom":
        report, err = generator.Generate(audit.PeriodTypeCustom, *fromDate, *toDate)
    default:
        log.Fatal().Msg("Invalid period type")
    }
    
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to generate report")
    }
    
    // Signature (si demandée)
    if *sign && jwsService != nil {
        if err := generator.Sign(report); err != nil {
            log.Warn().Err(err).Msg("Failed to sign report")
        }
    }
    
    // Export
    switch *format {
    case "json":
        if *outputPath == "" {
            *outputPath = fmt.Sprintf("report-%s.json", report.Period.StartDate)
        }
        if err := generator.ExportJSON(report, *outputPath); err != nil {
            log.Fatal().Err(err).Msg("Failed to export JSON")
        }
        log.Info().Str("path", *outputPath).Msg("Report exported to JSON")
        
    case "csv":
        if *outputPath == "" {
            *outputPath = fmt.Sprintf("report-%s.csv", report.Period.StartDate)
        }
        if err := generator.ExportCSV(report, *outputPath); err != nil {
            log.Fatal().Err(err).Msg("Failed to export CSV")
        }
        log.Info().Str("path", *outputPath).Msg("Report exported to CSV")
        
    case "pdf":
        if *outputPath == "" {
            *outputPath = fmt.Sprintf("report-%s.pdf", report.Period.StartDate)
        }
        pdfGen := audit.NewPDFGenerator(jwsService, *log)
        if err := pdfGen.Generate(report, *outputPath); err != nil {
            log.Fatal().Err(err).Msg("Failed to export PDF")
        }
        log.Info().Str("path", *outputPath).Msg("Report exported to PDF")
        
    default:
        log.Fatal().Str("format", *format).Msg("Invalid format")
    }
}
```

#### Compilation

```bash
# Compiler le binaire
go build -o bin/audit ./cmd/audit

# Ou avec version/commit
go build -ldflags "-X main.Version=$(git describe --tags) -X main.Commit=$(git rev-parse HEAD)" -o bin/audit ./cmd/audit
```

**Livrables J3** :
- ✅ `cmd/audit/main.go` (CLI complet)
- ✅ Tests unitaires `tests/unit/audit_cli_test.go` (10+ tests)
- ✅ Documentation usage dans README

---

### J4 : Tests & Documentation

#### Tests Unitaires

**1. `tests/unit/audit_report_test.go`** (15+ tests) :
- ✅ Test `GenerateMonthly` (données valides)
- ✅ Test `GenerateQuarterly` (données valides)
- ✅ Test `Generate` custom (données valides)
- ✅ Test collecte événements audit (logs existants)
- ✅ Test collecte statistiques documents (DB)
- ✅ Test collecte statistiques erreurs (logs)
- ✅ Test collecte statistiques performance (logs)
- ✅ Test collecte statistiques ledger (DB)
- ✅ Test collecte statistiques réconciliation (logs)
- ✅ Test collecte signatures journalières
- ✅ Test calcul résumé exécutif
- ✅ Test signature JWS rapport
- ✅ Test export JSON
- ✅ Test export CSV
- ✅ Test erreurs (période invalide, DB indisponible, etc.)

**2. `tests/unit/audit_pdf_test.go`** (10+ tests) :
- ✅ Test génération PDF (rapport valide)
- ✅ Test page de garde
- ✅ Test page résumé
- ✅ Test page statistiques documents
- ✅ Test page statistiques erreurs
- ✅ Test page performance
- ✅ Test page ledger
- ✅ Test page signatures
- ✅ Test QR code
- ✅ Test erreurs (rapport invalide, JWS indisponible)

**3. `tests/unit/audit_cli_test.go`** (10+ tests) :
- ✅ Test flags validation
- ✅ Test période monthly
- ✅ Test période quarterly
- ✅ Test période custom
- ✅ Test format json
- ✅ Test format csv
- ✅ Test format pdf
- ✅ Test signature JWS
- ✅ Test erreurs (flags invalides, fichiers manquants)

#### Tests d'Intégration

**1. Test end-to-end** :
```bash
# Générer rapport mensuel complet
./bin/audit --period monthly --year 2025 --month 1 --format json --sign --output /tmp/report-test.json

# Vérifier contenu JSON
cat /tmp/report-test.json | jq '.period.type'  # → "monthly"
cat /tmp/report-test.json | jq '.summary.total_documents'  # → nombre
cat /tmp/report-test.json | jq '.metadata.report_jws'  # → signature JWS

# Générer PDF
./bin/audit --period monthly --year 2025 --month 1 --format pdf --output /tmp/report-test.pdf

# Vérifier PDF généré
file /tmp/report-test.pdf  # → PDF document
```

#### Documentation

**1. `docs/audit_export_spec.md`** :
- Format rapport JSON (exemple complet)
- Format rapport CSV (exemple)
- Structure PDF (description pages)
- Signature JWS (format, vérification)
- Exemples d'utilisation CLI

**2. Mise à jour `README.md`** :
- Section "Génération Rapports d'Audit"
- Exemples CLI
- Configuration requise

**Livrables J4** :
- ✅ 35+ tests unitaires (100% réussite)
- ✅ Tests d'intégration
- ✅ Documentation `docs/audit_export_spec.md`
- ✅ Mise à jour README

---

## 🧪 Tests & Validation

### Checklist de Validation

#### Fonctionnalités

- [ ] Génération rapport mensuel (JSON, CSV, PDF)
- [ ] Génération rapport trimestriel (JSON, CSV, PDF)
- [ ] Génération rapport personnalisé (JSON, CSV, PDF)
- [ ] Signature JWS rapport
- [ ] Export JSON valide
- [ ] Export CSV valide
- [ ] Export PDF valide (8 pages)
- [ ] QR code hash dans PDF
- [ ] Collecte données logs audit
- [ ] Collecte données base de données
- [ ] Collecte signatures journalières
- [ ] Calcul statistiques correct
- [ ] CLI avec tous les flags

#### Performance

- [ ] Génération rapport 30 jours : < 10s
- [ ] Génération rapport 90 jours : < 30s
- [ ] Génération PDF : < 5s
- [ ] Signature JWS : < 1s

#### Qualité

- [ ] 35+ tests unitaires (100% réussite)
- [ ] 0 erreur de linter
- [ ] Documentation complète
- [ ] Exemples fonctionnels

---

## 📋 Livrables Finaux

| Type | Fichier | Description |
|:-----|:--------|:------------|
| **Go pkg** | `internal/audit/report.go` | Génération rapports JSON/CSV |
| **Go pkg** | `internal/audit/pdf.go` | Génération PDF avec template |
| **CLI** | `cmd/audit/main.go` | CLI génération rapports |
| **Tests** | `tests/unit/audit_report_test.go` | Tests module report (15+) |
| **Tests** | `tests/unit/audit_pdf_test.go` | Tests module PDF (10+) |
| **Tests** | `tests/unit/audit_cli_test.go` | Tests CLI (10+) |
| **Docs** | `docs/audit_export_spec.md` | Spécification format rapport |
| **Docs** | `README.md` | Mise à jour section rapports |

---

## 🔒 Sécurité

### Signature JWS

- **Hash** : SHA256 du rapport JSON (avant signature)
- **Signature** : RS256 avec clé privée JWS
- **Vérification** : Via JWKS public (`/jwks.json`)
- **Stockage** : Inclus dans `report.metadata.report_jws`

### Permissions Fichiers

- **Rapports générés** : Permissions `644` (lecture publique)
- **Rapports signés** : Inclure signature JWS dans métadonnées

### Validation

- **Période** : Vérifier `startDate <= endDate`
- **Dates** : Format `YYYY-MM-DD` strict
- **Output** : Vérifier répertoire accessible en écriture

---

## 🚀 Résultat Attendu

À la fin de la Phase 4.4, **Dorevia Vault** pourra :

✅ **Générer** des rapports d'audit consolidés (mensuels/trimestriels)  
✅ **Exporter** en JSON, CSV, PDF  
✅ **Signer** les rapports avec JWS pour preuve d'intégrité  
✅ **Automatiser** via CLI pour génération manuelle ou scriptable  
✅ **Conformer** aux exigences PDP/PPF 2026 (rapports auditable)

---

## 📅 Planning Détaillé

| Jour | Tâche | Durée | Livrables |
|:-----|:------|:------|:----------|
| **J1** | Module `report.go` | 8h | `internal/audit/report.go` + tests (15+) |
| **J2** | Module `pdf.go` | 8h | `internal/audit/pdf.go` + tests (10+) |
| **J3** | CLI `cmd/audit/main.go` | 6h | `cmd/audit/main.go` + tests (10+) |
| **J4** | Tests & Documentation | 2h | Documentation + validation complète |

**Total** : **24 heures** (4 jours ouvrés)

---

## ⚠️ Risques & Mitigation

| Risque | Impact | Probabilité | Mitigation |
|:-------|:-------|:------------|:------------|
| **Performance lente** (gros volumes) | 🟡 Moyen | Moyenne | Pagination, cache, optimisations requêtes |
| **PDF complexe** (gofpdf) | 🟡 Moyen | Faible | Template simple, tests progressifs |
| **Données manquantes** (DB indisponible) | 🟢 Faible | Faible | Mode dégradé (logs uniquement) |
| **Signature JWS échoue** | 🟢 Faible | Faible | Warning, rapport non signé |

---

## ✅ Critères de Succès

### Fonctionnels

- ✅ Génération rapport mensuel/trimestriel fonctionnelle
- ✅ Export JSON, CSV, PDF opérationnel
- ✅ Signature JWS intégrée
- ✅ CLI avec tous les flags
- ✅ 35+ tests unitaires (100% réussite)

### Techniques

- ✅ Performance acceptable (< 10s pour 30 jours)
- ✅ Documentation complète
- ✅ 0 erreur de linter
- ✅ Code maintenable et testé

### Conformité

- ✅ Format rapport conforme
- ✅ Signature JWS vérifiable
- ✅ Traçabilité complète (logs + DB + métriques)

---

**Document créé le** : Janvier 2025  
**Basé sur** : `docs/Dorevia_Vault_Sprint4.md` + `docs/ANALYSE_EXPERT_SPRINT4.md`  
**Version** : v1.2-dev (Sprint 4 Phase 4.4)

© 2025 Doreviateam | Projet Dorevia Vault — v1.2-dev

