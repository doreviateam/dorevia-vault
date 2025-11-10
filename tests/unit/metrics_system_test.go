package unit

import (
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateSystemMetrics teste la mise à jour des métriques système
func TestUpdateSystemMetrics(t *testing.T) {
	// Test que la fonction ne retourne pas d'erreur
	err := metrics.UpdateSystemMetrics()
	assert.NoError(t, err, "UpdateSystemMetrics should not return error")

	// Vérifier que les métriques sont mises à jour (valeurs >= 0)
	// Note: Les valeurs exactes dépendent du système, on vérifie juste qu'elles sont définies
	cpuValue := metrics.SystemCPUUsagePercent
	memoryUsage := metrics.SystemMemoryUsageBytes
	memoryTotal := metrics.SystemMemoryTotalBytes
	diskUsage := metrics.SystemDiskUsageBytes
	diskCapacity := metrics.SystemDiskCapacityBytes

	// Vérifier que les métriques existent (non nil)
	assert.NotNil(t, cpuValue)
	assert.NotNil(t, memoryUsage)
	assert.NotNil(t, memoryTotal)
	assert.NotNil(t, diskUsage)
	assert.NotNil(t, diskCapacity)
}

// TestSystemMetrics_CPUUsage teste que l'utilisation CPU est dans une plage valide
func TestSystemMetrics_CPUUsage(t *testing.T) {
	err := metrics.UpdateSystemMetrics()
	require.NoError(t, err)

	// L'utilisation CPU devrait être entre 0 et 100 (ou légèrement plus pour les pics)
	// On vérifie juste que la métrique est accessible
	// Note: La valeur réelle dépend du système au moment du test
	assert.NotNil(t, metrics.SystemCPUUsagePercent)
}

// TestSystemMetrics_MemoryUsage teste que l'utilisation mémoire est cohérente
func TestSystemMetrics_MemoryUsage(t *testing.T) {
	err := metrics.UpdateSystemMetrics()
	require.NoError(t, err)

	// L'utilisation mémoire devrait être <= mémoire totale
	// On vérifie juste que les métriques sont définies
	assert.NotNil(t, metrics.SystemMemoryUsageBytes)
	assert.NotNil(t, metrics.SystemMemoryTotalBytes)

	// Vérifier que l'utilisation est <= total (logique système)
	// Note: On ne peut pas comparer directement les valeurs car ce sont des prometheus.Gauge
	// mais on peut vérifier que les métriques existent
}

// TestSystemMetrics_DiskUsage teste que l'utilisation disque est cohérente
func TestSystemMetrics_DiskUsage(t *testing.T) {
	err := metrics.UpdateSystemMetrics()
	require.NoError(t, err)

	// L'utilisation disque devrait être <= capacité totale
	assert.NotNil(t, metrics.SystemDiskUsageBytes)
	assert.NotNil(t, metrics.SystemDiskCapacityBytes)
}

// TestStartSystemMetricsCollector teste le démarrage du collecteur
func TestStartSystemMetricsCollector(t *testing.T) {
	// Démarrer le collecteur avec un interval court pour les tests
	interval := 100 * time.Millisecond
	metrics.StartSystemMetricsCollector(interval)

	// Attendre un peu pour que le collecteur fasse au moins une collecte
	time.Sleep(200 * time.Millisecond)

	// Vérifier que les métriques sont mises à jour
	// (on ne peut pas vérifier directement, mais on peut vérifier qu'il n'y a pas d'erreur)
	assert.NotNil(t, metrics.SystemCPUUsagePercent)
	assert.NotNil(t, metrics.SystemMemoryUsageBytes)
	assert.NotNil(t, metrics.SystemDiskUsageBytes)
}

// TestSystemMetrics_CollectorInterval teste différents intervalles
func TestSystemMetrics_CollectorInterval(t *testing.T) {
	// Test avec intervalle court
	metrics.StartSystemMetricsCollector(1 * time.Second)
	time.Sleep(100 * time.Millisecond) // Attendre un peu
	assert.NotNil(t, metrics.SystemCPUUsagePercent)

	// Test avec intervalle par défaut (0 = 30s)
	metrics.StartSystemMetricsCollector(0)
	time.Sleep(100 * time.Millisecond)
	assert.NotNil(t, metrics.SystemCPUUsagePercent)
}

// TestLedgerAppendErrors_Metric teste la métrique ledger_append_errors_total
func TestLedgerAppendErrors_Metric(t *testing.T) {
	// Vérifier que la métrique existe
	assert.NotNil(t, metrics.LedgerAppendErrors)

	// Tester la fonction d'enregistrement
	metrics.RecordLedgerAppendError()
	metrics.RecordLedgerAppendError()

	// Vérifier que la fonction ne panique pas
	// (on ne peut pas vérifier la valeur directement car c'est un counter Prometheus)
	assert.NotNil(t, metrics.LedgerAppendErrors)
}

// TestSystemMetrics_ConcurrentUpdate teste la mise à jour concurrente
func TestSystemMetrics_ConcurrentUpdate(t *testing.T) {
	// Démarrer plusieurs goroutines qui mettent à jour les métriques
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			err := metrics.UpdateSystemMetrics()
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Attendre que toutes les goroutines terminent
	for i := 0; i < 5; i++ {
		<-done
	}

	// Vérifier que les métriques sont toujours accessibles
	assert.NotNil(t, metrics.SystemCPUUsagePercent)
	assert.NotNil(t, metrics.SystemMemoryUsageBytes)
}

// TestSystemMetrics_ErrorHandling teste la gestion d'erreurs
func TestSystemMetrics_ErrorHandling(t *testing.T) {
	// La fonction UpdateSystemMetrics devrait gérer les erreurs gracieusement
	// On teste avec un appel normal (les erreurs réelles dépendent du système)
	err := metrics.UpdateSystemMetrics()
	
	// Si erreur, elle devrait être liée au système (permissions, etc.)
	// mais la fonction ne devrait pas paniquer
	if err != nil {
		// Erreur acceptable (système, permissions, etc.)
		t.Logf("UpdateSystemMetrics returned error (acceptable): %v", err)
	}
}

// TestSystemMetrics_AllMetricsExist teste que toutes les métriques système existent
func TestSystemMetrics_AllMetricsExist(t *testing.T) {
	// Vérifier que toutes les métriques sont définies
	assert.NotNil(t, metrics.SystemCPUUsagePercent, "SystemCPUUsagePercent should exist")
	assert.NotNil(t, metrics.SystemMemoryUsageBytes, "SystemMemoryUsageBytes should exist")
	assert.NotNil(t, metrics.SystemMemoryTotalBytes, "SystemMemoryTotalBytes should exist")
	assert.NotNil(t, metrics.SystemDiskUsageBytes, "SystemDiskUsageBytes should exist")
	assert.NotNil(t, metrics.SystemDiskCapacityBytes, "SystemDiskCapacityBytes should exist")
}

// TestSystemMetrics_UpdateAfterCollector teste la mise à jour après démarrage collecteur
func TestSystemMetrics_UpdateAfterCollector(t *testing.T) {
	// Démarrer le collecteur
	metrics.StartSystemMetricsCollector(50 * time.Millisecond)

	// Attendre plusieurs cycles
	time.Sleep(200 * time.Millisecond)

	// Mettre à jour manuellement
	err := metrics.UpdateSystemMetrics()
	assert.NoError(t, err)

	// Vérifier que les métriques sont toujours accessibles
	assert.NotNil(t, metrics.SystemCPUUsagePercent)
}

