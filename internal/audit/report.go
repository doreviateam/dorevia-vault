package audit

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/doreviateam/dorevia-vault/internal/storage"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

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
	TotalDocuments       int64   `json:"total_documents"`        // Total documents vaultés
	TotalErrors          int64   `json:"total_errors"`           // Total erreurs
	ErrorRate            float64 `json:"error_rate"`             // Taux d'erreur (%)
	TotalLedgerEntries   int64   `json:"total_ledger_entries"`   // Total entrées ledger
	TotalReconciliations  int64   `json:"total_reconciliations"`  // Total réconciliations
	AvgDocumentSize      int64   `json:"avg_document_size"`      // Taille moyenne document (bytes)
	TotalStorageSize     int64   `json:"total_storage_size"`     // Taille totale stockage (bytes)
}

// DocumentStats représente les statistiques sur les documents
type DocumentStats struct {
	Total            int64             `json:"total"`             // Total documents
	ByStatus         map[string]int64  `json:"by_status"`        // Par statut (success, error, idempotent)
	BySource         map[string]int64  `json:"by_source"`         // Par source (sales, purchase, pos, etc.)
	ByContentType    map[string]int64  `json:"by_content_type"`   // Par type MIME
	SizeDistribution SizeDistribution  `json:"size_distribution"` // Distribution des tailles
}

// SizeDistribution représente la distribution des tailles de documents
type SizeDistribution struct {
	Min    int64   `json:"min"`     // Taille minimale (bytes)
	Max    int64   `json:"max"`     // Taille maximale (bytes)
	Mean   float64 `json:"mean"`    // Taille moyenne (bytes)
	Median int64   `json:"median"`   // Taille médiane (bytes)
	P95    int64   `json:"p95"`     // Percentile 95 (bytes)
	P99    int64   `json:"p99"`     // Percentile 99 (bytes)
}

// ErrorStats représente les statistiques sur les erreurs
type ErrorStats struct {
	Total          int64          `json:"total"`           // Total erreurs
	ByType         map[string]int64 `json:"by_type"`      // Par type d'erreur
	ByEventType    map[string]int64 `json:"by_event_type"` // Par type d'événement
	CriticalErrors []CriticalError `json:"critical_errors"` // Erreurs critiques (top 10)
}

// CriticalError représente une erreur critique
type CriticalError struct {
	Timestamp  string `json:"timestamp"`   // RFC3339
	EventType  string `json:"event_type"`  // Type d'événement
	DocumentID string `json:"document_id"` // ID document (si applicable)
	Message    string `json:"message"`     // Message d'erreur
	Count      int64  `json:"count"`       // Nombre d'occurrences
}

// PerformanceStats représente les statistiques de performance
type PerformanceStats struct {
	DocumentStorage PerformanceMetric `json:"document_storage"` // Stockage documents
	JWSSignature    PerformanceMetric `json:"jws_signature"`    // Signature JWS
	LedgerAppend    PerformanceMetric `json:"ledger_append"`    // Ajout ledger
	Transaction     PerformanceMetric `json:"transaction"`      // Transactions
}

// PerformanceMetric représente une métrique de performance
type PerformanceMetric struct {
	Count  int64   `json:"count"`  // Nombre d'observations
	Mean   float64 `json:"mean"`  // Durée moyenne (secondes)
	Median float64 `json:"median"` // Durée médiane (secondes)
	P50    float64 `json:"p50"`    // Percentile 50 (secondes)
	P95    float64 `json:"p95"`    // Percentile 95 (secondes)
	P99    float64 `json:"p99"`    // Percentile 99 (secondes)
	Min    float64 `json:"min"`    // Durée minimale (secondes)
	Max    float64 `json:"max"`    // Durée maximale (secondes)
}

// LedgerStats représente les statistiques sur le ledger
type LedgerStats struct {
	TotalEntries   int64   `json:"total_entries"`   // Total entrées
	NewEntries     int64   `json:"new_entries"`     // Nouvelles entrées (période)
	Errors         int64   `json:"errors"`          // Erreurs ledger
	ErrorRate      float64 `json:"error_rate"`      // Taux d'erreur (%)
	CurrentSize    int64   `json:"current_size"`    // Taille actuelle
	ChainIntegrity bool    `json:"chain_integrity"` // Intégrité chaîne (vérifiée)
	LastHash       string  `json:"last_hash"`       // Dernier hash
}

// ReconciliationStats représente les statistiques sur les réconciliations
type ReconciliationStats struct {
	TotalRuns        int64 `json:"total_runs"`         // Total exécutions
	SuccessfulRuns   int64 `json:"successful_runs"`    // Exécutions réussies
	FailedRuns       int64 `json:"failed_runs"`        // Exécutions échouées
	OrphanFilesFound int64 `json:"orphan_files_found"` // Fichiers orphelins trouvés
	OrphanFilesFixed int64 `json:"orphan_files_fixed"` // Fichiers orphelins corrigés
	DocumentsFixed   int64 `json:"documents_fixed"`   // Documents corrigés
}

// ReportMetadata représente les métadonnées du rapport
type ReportMetadata struct {
	GeneratedAt string   `json:"generated_at"`  // RFC3339
	GeneratedBy string   `json:"generated_by"`  // "dorevia-vault" ou "cli"
	Version     string   `json:"version"`      // Version du système
	ReportID    string   `json:"report_id"`    // UUID unique du rapport
	ReportHash  string   `json:"report_hash"`  // SHA256 du rapport JSON
	ReportJWS   string   `json:"report_jws"`   // Signature JWS du rapport (si signé)
	DataSources []string `json:"data_sources"` // Sources de données utilisées
}

// AuditReport représente le rapport d'audit complet
type AuditReport struct {
	Period         Period              `json:"period"`
	Summary        ReportSummary       `json:"summary"`
	Documents      DocumentStats       `json:"documents"`
	Errors         ErrorStats          `json:"errors"`
	Performance    PerformanceStats    `json:"performance"`
	Ledger         LedgerStats         `json:"ledger"`
	Reconciliation ReconciliationStats `json:"reconciliation"`
	Signatures     []DailyHash         `json:"signatures"` // Signatures journalières de la période
	Metadata       ReportMetadata      `json:"metadata"`
}

// ReportGenerator génère des rapports d'audit
type ReportGenerator struct {
	logger     *Logger
	exporter   *Exporter
	db         *storage.DB // Optionnel (si DB disponible)
	jwsService *crypto.Service
	log        zerolog.Logger
	version    string // Version du système
}

// NewReportGenerator crée un nouveau générateur de rapports
func NewReportGenerator(logger *Logger, exporter *Exporter, db *storage.DB, jwsService *crypto.Service, log zerolog.Logger) *ReportGenerator {
	return &ReportGenerator{
		logger:     logger,
		exporter:   exporter,
		db:         db,
		jwsService: jwsService,
		log:        log,
		version:    "v1.2.0-rc1", // TODO: injecter depuis config
	}
}

// Generate génère un rapport pour une période donnée
func (g *ReportGenerator) Generate(periodType PeriodType, startDate, endDate string) (*AuditReport, error) {
	// Valider les dates
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}
	if start.After(end) {
		return nil, fmt.Errorf("start date must be before or equal to end date")
	}

	// Créer la période
	period := Period{
		Type:      periodType,
		StartDate: startDate,
		EndDate:   endDate,
		Label:     formatPeriodLabel(periodType, start, end),
	}

	// Collecter toutes les données
	events, err := g.collectAuditEvents(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to collect audit events: %w", err)
	}

	docStats, err := g.collectDocumentStats(startDate, endDate)
	if err != nil {
		g.log.Warn().Err(err).Msg("Failed to collect document stats, using empty stats")
		docStats = &DocumentStats{
			Total:         0,
			ByStatus:      make(map[string]int64),
			BySource:      make(map[string]int64),
			ByContentType: make(map[string]int64),
			SizeDistribution: SizeDistribution{},
		}
	}

	errorStats := g.collectErrorStats(events)
	perfStats := g.collectPerformanceStats(events)
	reconStats := g.collectReconciliationStats(events)

	ledgerStats, err := g.collectLedgerStats(startDate, endDate)
	if err != nil {
		g.log.Warn().Err(err).Msg("Failed to collect ledger stats, using empty stats")
		ledgerStats = &LedgerStats{
			TotalEntries:   0,
			NewEntries:     0,
			Errors:          0,
			ErrorRate:       0,
			CurrentSize:     0,
			ChainIntegrity:  false,
			LastHash:        "",
		}
	}

	signatures, err := g.collectDailySignatures(startDate, endDate)
	if err != nil {
		g.log.Warn().Err(err).Msg("Failed to collect daily signatures, continuing without signatures")
		signatures = []DailyHash{}
	}

	// Calculer le résumé
	summary := g.calculateSummary(docStats, errorStats, ledgerStats, reconStats)

	// Créer les métadonnées
	reportID := uuid.New().String()
	dataSources := []string{"audit_logs"}
	if g.db != nil {
		dataSources = append(dataSources, "database")
	}
	if len(signatures) > 0 {
		dataSources = append(dataSources, "daily_signatures")
	}

	metadata := ReportMetadata{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GeneratedBy: "cli",
		Version:     g.version,
		ReportID:    reportID,
		ReportHash:  "", // Sera calculé lors de l'export
		ReportJWS:   "", // Sera ajouté lors de la signature
		DataSources: dataSources,
	}

	// Créer le rapport
	report := &AuditReport{
		Period:         period,
		Summary:        *summary,
		Documents:      *docStats,
		Errors:         *errorStats,
		Performance:    *perfStats,
		Ledger:         *ledgerStats,
		Reconciliation: *reconStats,
		Signatures:     signatures,
		Metadata:       metadata,
	}

	return report, nil
}

// GenerateMonthly génère un rapport mensuel
func (g *ReportGenerator) GenerateMonthly(year int, month int) (*AuditReport, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("invalid month: %d (must be 1-12)", month)
	}

	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 1, 0).AddDate(0, 0, -1) // Dernier jour du mois

	return g.Generate(PeriodTypeMonthly, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

// GenerateQuarterly génère un rapport trimestriel
func (g *ReportGenerator) GenerateQuarterly(year int, quarter int) (*AuditReport, error) {
	if quarter < 1 || quarter > 4 {
		return nil, fmt.Errorf("invalid quarter: %d (must be 1-4)", quarter)
	}

	// Calculer le mois de début du trimestre
	startMonth := time.Month((quarter-1)*3 + 1)
	startDate := time.Date(year, startMonth, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 3, 0).AddDate(0, 0, -1) // Dernier jour du trimestre

	return g.Generate(PeriodTypeQuarterly, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

// collectAuditEvents collecte les événements d'audit depuis les logs
func (g *ReportGenerator) collectAuditEvents(startDate, endDate string) ([]Event, error) {
	// Utiliser Exporter pour récupérer tous les événements
	opts := ExportOptions{
		From:   startDate,
		To:     endDate,
		Page:   1,
		Limit:  10000, // Max pour récupérer tout
		Format: ExportFormatJSON,
	}

	result, err := g.exporter.Export(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to export audit events: %w", err)
	}

	// Si plusieurs pages, récupérer toutes les pages
	allEvents := result.Events
	for result.HasNext {
		opts.Page++
		result, err = g.exporter.Export(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to export audit events page %d: %w", opts.Page, err)
		}
		allEvents = append(allEvents, result.Events...)
	}

	return allEvents, nil
}

// collectDocumentStats collecte les statistiques documents depuis la DB
func (g *ReportGenerator) collectDocumentStats(startDate, endDate string) (*DocumentStats, error) {
	if g.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	ctx := context.Background()

	// Convertir les dates en timestamps
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour).Add(-1 * time.Second) // Fin de journée

	// Requête pour statistiques par statut et source
	query := `
		SELECT 
			COUNT(*) as total,
			COALESCE(source, 'unknown') as source,
			COALESCE(content_type, 'unknown') as content_type,
			AVG(size_bytes) as avg_size,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY size_bytes) as median,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY size_bytes) as p95,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY size_bytes) as p99,
			MIN(size_bytes) as min_size,
			MAX(size_bytes) as max_size
		FROM documents
		WHERE created_at >= $1 AND created_at <= $2
		GROUP BY source, content_type
	`

	rows, err := g.db.Pool.Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	stats := &DocumentStats{
		Total:         0,
		ByStatus:      make(map[string]int64),
		BySource:      make(map[string]int64),
		ByContentType: make(map[string]int64),
		SizeDistribution: SizeDistribution{},
	}

	var sizes []int64
	bySource := make(map[string]int64)
	byContentType := make(map[string]int64)
	var totalSize int64

	for rows.Next() {
		var source, contentType string
		var total, avgSize, median, p95, p99, minSize, maxSize *float64

		err := rows.Scan(&total, &source, &contentType, &avgSize, &median, &p95, &p99, &minSize, &maxSize)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		if total != nil {
			count := int64(*total)
			stats.Total += count
			bySource[source] += count
			byContentType[contentType] += count

			if avgSize != nil {
				totalSize += int64(*avgSize) * count
			}
		}
	}

	// Requête pour récupérer toutes les tailles pour calculer les percentiles
	sizeQuery := `
		SELECT size_bytes
		FROM documents
		WHERE created_at >= $1 AND created_at <= $2 AND size_bytes IS NOT NULL
		ORDER BY size_bytes
	`

	sizeRows, err := g.db.Pool.Query(ctx, sizeQuery, start, end)
	if err == nil {
		defer sizeRows.Close()
		for sizeRows.Next() {
			var size int64
			if err := sizeRows.Scan(&size); err == nil {
				sizes = append(sizes, size)
			}
		}
	}

	// Calculer la distribution des tailles
	if len(sizes) > 0 {
		stats.SizeDistribution = calculateSizeDistribution(sizes)
	}

	stats.BySource = bySource
	stats.ByContentType = byContentType

	// Calculer la taille moyenne
	if stats.Total > 0 {
		stats.SizeDistribution.Mean = float64(totalSize) / float64(stats.Total)
	}

	return stats, nil
}

// collectErrorStats collecte les statistiques d'erreurs depuis les logs
func (g *ReportGenerator) collectErrorStats(events []Event) *ErrorStats {
	stats := &ErrorStats{
		Total:          0,
		ByType:         make(map[string]int64),
		ByEventType:    make(map[string]int64),
		CriticalErrors: []CriticalError{},
	}

	errorMap := make(map[string]*CriticalError)

	for _, event := range events {
		if event.Status == EventStatusError {
			stats.Total++
			stats.ByEventType[string(event.EventType)]++

			// Extraire le type d'erreur depuis les métadonnées
			errorType := "unknown"
			if msg, ok := event.Metadata["error"].(string); ok {
				errorType = msg
			} else if msg, ok := event.Metadata["message"].(string); ok {
				errorType = msg
			}

			stats.ByType[errorType]++

			// Collecter les erreurs critiques
			key := fmt.Sprintf("%s:%s", event.EventType, errorType)
			if err, exists := errorMap[key]; exists {
				err.Count++
			} else {
				errorMap[key] = &CriticalError{
					Timestamp:  event.Timestamp,
					EventType:  string(event.EventType),
					DocumentID: event.DocumentID,
					Message:    errorType,
					Count:      1,
				}
			}
		}
	}

	// Trier les erreurs critiques par nombre d'occurrences (top 10)
	criticalErrors := make([]CriticalError, 0, len(errorMap))
	for _, err := range errorMap {
		criticalErrors = append(criticalErrors, *err)
	}
	sort.Slice(criticalErrors, func(i, j int) bool {
		return criticalErrors[i].Count > criticalErrors[j].Count
	})
	if len(criticalErrors) > 10 {
		criticalErrors = criticalErrors[:10]
	}
	stats.CriticalErrors = criticalErrors

	return stats
}

// collectPerformanceStats collecte les statistiques de performance depuis les logs
func (g *ReportGenerator) collectPerformanceStats(events []Event) *PerformanceStats {
	stats := &PerformanceStats{
		DocumentStorage: PerformanceMetric{},
		JWSSignature:     PerformanceMetric{},
		LedgerAppend:     PerformanceMetric{},
		Transaction:      PerformanceMetric{},
	}

	// Extraire les durées par type d'événement
	storageDurations := []float64{}
	jwsDurations := []float64{}
	ledgerDurations := []float64{}
	transactionDurations := []float64{}

	for _, event := range events {
		if event.DurationMS > 0 {
			durationSec := float64(event.DurationMS) / 1000.0

			switch event.EventType {
			case EventTypeDocumentVaulted:
				storageDurations = append(storageDurations, durationSec)
			case EventTypeJWSSigned:
				jwsDurations = append(jwsDurations, durationSec)
			case EventTypeLedgerAppended:
				ledgerDurations = append(ledgerDurations, durationSec)
			}

			// Transactions incluent tous les événements avec durée
			transactionDurations = append(transactionDurations, durationSec)
		}
	}

	stats.DocumentStorage = calculatePerformanceMetric(storageDurations)
	stats.JWSSignature = calculatePerformanceMetric(jwsDurations)
	stats.LedgerAppend = calculatePerformanceMetric(ledgerDurations)
	stats.Transaction = calculatePerformanceMetric(transactionDurations)

	return stats
}

// collectLedgerStats collecte les statistiques ledger depuis la DB
func (g *ReportGenerator) collectLedgerStats(startDate, endDate string) (*LedgerStats, error) {
	if g.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	ctx := context.Background()

	// Convertir les dates en timestamps
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)
	end = end.Add(24 * time.Hour).Add(-1 * time.Second) // Fin de journée

	// Statistiques ledger pour la période
	query := `
		SELECT 
			COUNT(*) as total_entries,
			COUNT(CASE WHEN evidence_jws IS NULL OR evidence_jws = '' THEN 1 END) as errors
		FROM ledger
		WHERE timestamp >= $1 AND timestamp <= $2
	`

	var totalEntries, errors int64
	err := g.db.Pool.QueryRow(ctx, query, start, end).Scan(&totalEntries, &errors)
	if err != nil {
		return nil, fmt.Errorf("failed to query ledger stats: %w", err)
	}

	// Taille actuelle du ledger
	var currentSize int64
	sizeQuery := `SELECT COUNT(*) FROM ledger`
	err = g.db.Pool.QueryRow(ctx, sizeQuery).Scan(&currentSize)
	if err != nil {
		currentSize = 0
	}

	// Dernier hash
	var lastHash string
	lastHashQuery := `SELECT hash FROM ledger ORDER BY timestamp DESC LIMIT 1`
	err = g.db.Pool.QueryRow(ctx, lastHashQuery).Scan(&lastHash)
	if err != nil {
		lastHash = ""
	}

	// Vérification intégrité chaîne (simplifiée : vérifier que previous_hash correspond)
	integrityQuery := `
		SELECT COUNT(*)
		FROM ledger l1
		LEFT JOIN ledger l2 ON l2.hash = l1.previous_hash
		WHERE l1.previous_hash IS NOT NULL AND l1.previous_hash != ''
		AND l1.timestamp >= $1 AND l1.timestamp <= $2
		AND l2.hash IS NULL
	`

	var brokenLinks int64
	err = g.db.Pool.QueryRow(ctx, integrityQuery, start, end).Scan(&brokenLinks)
	chainIntegrity := err == nil && brokenLinks == 0

	errorRate := float64(0)
	if totalEntries > 0 {
		errorRate = float64(errors) / float64(totalEntries) * 100
	}

	return &LedgerStats{
		TotalEntries:   totalEntries,
		NewEntries:     totalEntries, // Pour la période, toutes sont nouvelles
		Errors:         errors,
		ErrorRate:      errorRate,
		CurrentSize:    currentSize,
		ChainIntegrity: chainIntegrity,
		LastHash:       lastHash,
	}, nil
}

// collectReconciliationStats collecte les statistiques réconciliation depuis les logs
func (g *ReportGenerator) collectReconciliationStats(events []Event) *ReconciliationStats {
	stats := &ReconciliationStats{
		TotalRuns:        0,
		SuccessfulRuns:   0,
		FailedRuns:        0,
		OrphanFilesFound: 0,
		OrphanFilesFixed: 0,
		DocumentsFixed:   0,
	}

	for _, event := range events {
		if event.EventType == EventTypeReconciliationRun {
			stats.TotalRuns++

			if event.Status == EventStatusSuccess {
				stats.SuccessfulRuns++
			} else if event.Status == EventStatusError {
				stats.FailedRuns++
			}

			// Extraire les métadonnées de réconciliation
			if metadata := event.Metadata; metadata != nil {
				if found, ok := metadata["orphan_files_found"].(float64); ok {
					stats.OrphanFilesFound += int64(found)
				}
				if fixed, ok := metadata["orphan_files_fixed"].(float64); ok {
					stats.OrphanFilesFixed += int64(fixed)
				}
				if docsFixed, ok := metadata["documents_fixed"].(float64); ok {
					stats.DocumentsFixed += int64(docsFixed)
				}
			}
		}
	}

	return stats
}

// collectDailySignatures collecte les signatures journalières
func (g *ReportGenerator) collectDailySignatures(startDate, endDate string) ([]DailyHash, error) {
	start, _ := time.Parse("2006-01-02", startDate)
	end, _ := time.Parse("2006-01-02", endDate)

	signatures := []DailyHash{}
	currentDate := start

	for !currentDate.After(end) {
		dateStr := currentDate.Format("2006-01-02")
		sigPath := g.logger.GetSignaturePath(dateStr)

		// Lire le fichier de signature s'il existe
		if data, err := os.ReadFile(sigPath); err == nil {
			var dailyHash DailyHash
			if err := json.Unmarshal(data, &dailyHash); err == nil {
				signatures = append(signatures, dailyHash)
			}
		}

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return signatures, nil
}

// calculateSummary calcule le résumé exécutif
func (g *ReportGenerator) calculateSummary(docs *DocumentStats, errors *ErrorStats, ledger *LedgerStats, recon *ReconciliationStats) *ReportSummary {
	errorRate := float64(0)
	if docs.Total > 0 {
		errorRate = float64(errors.Total) / float64(docs.Total) * 100
	}

	avgSize := int64(0)
	if docs.Total > 0 && docs.SizeDistribution.Mean > 0 {
		avgSize = int64(docs.SizeDistribution.Mean)
	}

	totalStorageSize := int64(0)
	if docs.Total > 0 {
		totalStorageSize = avgSize * docs.Total
	}

	return &ReportSummary{
		TotalDocuments:      docs.Total,
		TotalErrors:          errors.Total,
		ErrorRate:            errorRate,
		TotalLedgerEntries:   ledger.NewEntries,
		TotalReconciliations: recon.TotalRuns,
		AvgDocumentSize:     avgSize,
		TotalStorageSize:     totalStorageSize,
	}
}

// Sign signe le rapport avec JWS
func (g *ReportGenerator) Sign(report *AuditReport) error {
	if g.jwsService == nil {
		return fmt.Errorf("JWS service not available")
	}

	// Calculer le hash du rapport JSON (sans la signature)
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	hash := sha256.Sum256(reportJSON)
	hashHex := hex.EncodeToString(hash[:])

	// Signer le hash
	evidence, err := g.jwsService.SignEvidence(
		report.Metadata.ReportID,
		hashHex,
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to sign report: %w", err)
	}

	// Mettre à jour les métadonnées
	report.Metadata.ReportHash = hashHex
	report.Metadata.ReportJWS = evidence

	return nil
}

// ExportJSON exporte le rapport en JSON
func (g *ReportGenerator) ExportJSON(report *AuditReport, outputPath string) error {
	// Calculer le hash si pas déjà fait
	if report.Metadata.ReportHash == "" {
		reportJSON, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("failed to marshal report: %w", err)
		}
		hash := sha256.Sum256(reportJSON)
		report.Metadata.ReportHash = hex.EncodeToString(hash[:])
	}

	// Marshal le rapport
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	// Écrire dans le fichier
	if outputPath == "" || outputPath == "-" {
		// stdout
		fmt.Println(string(reportJSON))
		return nil
	}

	return os.WriteFile(outputPath, reportJSON, 0644)
}

// ExportCSV exporte le rapport en CSV (format simplifié)
func (g *ReportGenerator) ExportCSV(report *AuditReport, outputPath string) error {
	file := os.Stdout
	if outputPath != "" && outputPath != "-" {
		var err error
		file, err = os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create CSV file: %w", err)
		}
		defer file.Close()
	}

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// En-têtes
	headers := []string{
		"period_type", "period_start", "period_end",
		"total_documents", "total_errors", "error_rate",
		"avg_document_size", "total_storage_size",
		"total_ledger_entries", "total_reconciliations",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Données
	row := []string{
		string(report.Period.Type),
		report.Period.StartDate,
		report.Period.EndDate,
		fmt.Sprintf("%d", report.Summary.TotalDocuments),
		fmt.Sprintf("%d", report.Summary.TotalErrors),
		fmt.Sprintf("%.2f", report.Summary.ErrorRate),
		fmt.Sprintf("%d", report.Summary.AvgDocumentSize),
		fmt.Sprintf("%d", report.Summary.TotalStorageSize),
		fmt.Sprintf("%d", report.Summary.TotalLedgerEntries),
		fmt.Sprintf("%d", report.Summary.TotalReconciliations),
	}
	if err := writer.Write(row); err != nil {
		return fmt.Errorf("failed to write CSV row: %w", err)
	}

	return nil
}

// Helper functions

func formatPeriodLabel(periodType PeriodType, start, end time.Time) string {
	switch periodType {
	case PeriodTypeMonthly:
		return start.Format("Janvier 2006")
	case PeriodTypeQuarterly:
		quarter := (int(start.Month())-1)/3 + 1
		return fmt.Sprintf("Q%d %d", quarter, start.Year())
	case PeriodTypeCustom:
		return fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	default:
		return fmt.Sprintf("%s - %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
}

func calculateSizeDistribution(sizes []int64) SizeDistribution {
	if len(sizes) == 0 {
		return SizeDistribution{}
	}

	sort.Slice(sizes, func(i, j int) bool {
		return sizes[i] < sizes[j]
	})

	n := len(sizes)
	dist := SizeDistribution{
		Min:    sizes[0],
		Max:    sizes[n-1],
		Median: sizes[n/2],
		P95:    sizes[int(float64(n)*0.95)],
		P99:    sizes[int(float64(n)*0.99)],
	}

	// Calculer la moyenne
	var sum int64
	for _, s := range sizes {
		sum += s
	}
	dist.Mean = float64(sum) / float64(n)

	return dist
}

func calculatePerformanceMetric(durations []float64) PerformanceMetric {
	if len(durations) == 0 {
		return PerformanceMetric{}
	}

	sort.Float64s(durations)
	n := len(durations)

	metric := PerformanceMetric{
		Count:  int64(n),
		Min:    durations[0],
		Max:    durations[n-1],
		Median: durations[n/2],
		P50:    durations[n/2],
		P95:    durations[int(float64(n)*0.95)],
		P99:    durations[int(float64(n)*0.99)],
	}

	// Calculer la moyenne
	var sum float64
	for _, d := range durations {
		sum += d
	}
	metric.Mean = sum / float64(n)

	return metric
}

