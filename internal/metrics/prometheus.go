package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Métriques métier Dorevia Vault (Sprint 3 Phase 2+)
// Toutes les métriques sont enregistrées automatiquement via promauto

var (
	// ============================================
	// COUNTERS - Compteurs d'événements
	// ============================================

	// DocumentsVaulted compte le nombre total de documents vaultés
	// Labels:
	//   - status: "success" | "error" | "idempotent"
	//   - source: "sales" | "purchase" | "pos" | "stock" | "sale" | "unknown"
	DocumentsVaulted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "documents_vaulted_total",
			Help: "Nombre total de documents vaultés par statut et source",
		},
		[]string{"status", "source"},
	)

	// JWSSignatures compte le nombre total de signatures JWS générées
	// Labels:
	//   - status: "success" | "error" | "degraded"
	JWSSignatures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jws_signatures_total",
			Help: "Nombre total de signatures JWS générées par statut",
		},
		[]string{"status"},
	)

	// LedgerEntries compte le nombre total d'entrées ajoutées au ledger
	LedgerEntries = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ledger_entries_total",
			Help: "Nombre total d'entrées ajoutées au ledger",
		},
	)

	// LedgerAppendErrors compte le nombre total d'erreurs lors de l'ajout au ledger
	// Sprint 4 Phase 4.1 : Nouvelle métrique pour détecter les erreurs ledger
	LedgerAppendErrors = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ledger_append_errors_total",
			Help: "Nombre total d'erreurs lors de l'ajout au ledger",
		},
	)

	// ReconciliationRuns compte le nombre total d'exécutions de réconciliation
	// Labels:
	//   - status: "success" | "error"
	ReconciliationRuns = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "reconciliation_runs_total",
			Help: "Nombre total d'exécutions de réconciliation par statut",
		},
		[]string{"status"},
	)

	// ============================================
	// HISTOGRAMMES - Durées d'opérations
	// ============================================

	// DocumentStorageDuration mesure la durée de stockage des documents
	// Labels:
	//   - operation: "store" | "verify"
	// Buckets: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 secondes
	DocumentStorageDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "document_storage_duration_seconds",
			Help:    "Durée de stockage des documents en secondes",
			Buckets: prometheus.ExponentialBuckets(0.001, 2.5, 12), // 1ms à ~10s
		},
		[]string{"operation"},
	)

	// JWSSignatureDuration mesure la durée de génération des signatures JWS
	// Buckets: 0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1 seconde
	JWSSignatureDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "jws_signature_duration_seconds",
			Help:    "Durée de génération des signatures JWS en secondes",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2.5, 12), // 0.1ms à ~1s
		},
	)

	// LedgerAppendDuration mesure la durée d'ajout d'entrées au ledger
	// Buckets: 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10 secondes
	LedgerAppendDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ledger_append_duration_seconds",
			Help:    "Durée d'ajout d'entrées au ledger en secondes",
			Buckets: prometheus.ExponentialBuckets(0.001, 2.5, 12), // 1ms à ~10s
		},
	)

	// TransactionDuration mesure la durée totale des transactions
	// Buckets: 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 25, 30 secondes
	TransactionDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "transaction_duration_seconds",
			Help:    "Durée totale des transactions en secondes",
			Buckets: prometheus.ExponentialBuckets(0.01, 2.5, 12), // 10ms à ~30s
		},
	)

	// ============================================
	// GAUGES - Valeurs instantanées
	// ============================================

	// LedgerSize mesure la taille actuelle du ledger (nombre d'entrées)
	LedgerSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ledger_size",
			Help: "Nombre actuel d'entrées dans le ledger",
		},
	)

	// StorageSizeBytes mesure la taille totale du stockage en octets
	StorageSizeBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "storage_size_bytes",
			Help: "Taille totale du stockage de documents en octets",
		},
	)

	// ActiveConnections mesure le nombre de connexions actives à la base de données
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "Nombre de connexions actives à la base de données",
		},
	)
)

// Helper functions pour faciliter l'utilisation des métriques

// RecordDocumentVaulted enregistre un document vaulté
func RecordDocumentVaulted(status, source string) {
	// Normaliser la source (tolower pour cohérence)
	sourceNormalized := normalizeSource(source)
	DocumentsVaulted.WithLabelValues(status, sourceNormalized).Inc()
}

// RecordJWSSignature enregistre une signature JWS
func RecordJWSSignature(status string) {
	JWSSignatures.WithLabelValues(status).Inc()
}

// RecordReconciliationRun enregistre une exécution de réconciliation
func RecordReconciliationRun(status string) {
	ReconciliationRuns.WithLabelValues(status).Inc()
}

// RecordLedgerAppendError enregistre une erreur lors de l'ajout au ledger
// Sprint 4 Phase 4.1 : Nouvelle fonction pour enregistrer les erreurs ledger
func RecordLedgerAppendError() {
	LedgerAppendErrors.Inc()
}

// RecordDocumentStorageDuration enregistre la durée de stockage d'un document
func RecordDocumentStorageDuration(operation string, durationSeconds float64) {
	DocumentStorageDuration.WithLabelValues(operation).Observe(durationSeconds)
}

// RecordJWSSignatureDuration enregistre la durée de génération d'une signature JWS
func RecordJWSSignatureDuration(durationSeconds float64) {
	JWSSignatureDuration.Observe(durationSeconds)
}

// RecordLedgerAppendDuration enregistre la durée d'ajout au ledger
func RecordLedgerAppendDuration(durationSeconds float64) {
	LedgerAppendDuration.Observe(durationSeconds)
}

// RecordTransactionDuration enregistre la durée totale d'une transaction
func RecordTransactionDuration(durationSeconds float64) {
	TransactionDuration.Observe(durationSeconds)
}

// UpdateLedgerSize met à jour la taille du ledger
func UpdateLedgerSize(size int64) {
	LedgerSize.Set(float64(size))
}

// UpdateStorageSizeBytes met à jour la taille du stockage
func UpdateStorageSizeBytes(sizeBytes int64) {
	StorageSizeBytes.Set(float64(sizeBytes))
}

// UpdateActiveConnections met à jour le nombre de connexions actives
func UpdateActiveConnections(count int) {
	ActiveConnections.Set(float64(count))
}

// normalizeSource normalise la source pour cohérence des métriques
// Sources possibles: sales, purchase, pos, stock, sale, unknown
func normalizeSource(source string) string {
	if source == "" {
		return "unknown"
	}
	// Convertir en minuscules pour cohérence
	normalized := source
	if len(normalized) > 0 {
		// Sources valides selon le code: sales, purchase, pos, stock, sale
		switch normalized {
		case "sales", "purchase", "pos", "stock", "sale":
			return normalized
		default:
			return "unknown"
		}
	}
	return "unknown"
}

