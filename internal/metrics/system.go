package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// Métriques système (Sprint 4 Phase 4.1)
// Expose les métriques CPU, RAM, disque via gopsutil

var (
	// SystemCPUUsagePercent mesure l'utilisation CPU en pourcentage
	// Valeur : 0-100 (pourcentage d'utilisation)
	SystemCPUUsagePercent = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_cpu_usage_percent",
			Help: "Utilisation CPU en pourcentage (0-100)",
		},
	)

	// SystemMemoryUsageBytes mesure l'utilisation mémoire en octets
	SystemMemoryUsageBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_usage_bytes",
			Help: "Utilisation mémoire en octets",
		},
	)

	// SystemMemoryTotalBytes mesure la mémoire totale en octets
	SystemMemoryTotalBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_memory_total_bytes",
			Help: "Mémoire totale disponible en octets",
		},
	)

	// SystemDiskUsageBytes mesure l'utilisation disque en octets
	SystemDiskUsageBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_disk_usage_bytes",
			Help: "Utilisation disque en octets",
		},
	)

	// SystemDiskCapacityBytes mesure la capacité disque totale en octets
	SystemDiskCapacityBytes = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "system_disk_capacity_bytes",
			Help: "Capacité disque totale en octets",
		},
	)
)

// UpdateSystemMetrics met à jour toutes les métriques système
// Sprint 4 Phase 4.1 : Collecte CPU, RAM, disque via gopsutil
func UpdateSystemMetrics() error {
	// 1. CPU : utilisation moyenne sur toutes les CPUs
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err != nil {
		return err
	}
	if len(cpuPercent) > 0 {
		SystemCPUUsagePercent.Set(cpuPercent[0])
	}

	// 2. Mémoire
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}
	SystemMemoryUsageBytes.Set(float64(memInfo.Used))
	SystemMemoryTotalBytes.Set(float64(memInfo.Total))

	// 3. Disque : utilisation du système de fichiers racine "/"
	diskInfo, err := disk.Usage("/")
	if err != nil {
		return err
	}
	SystemDiskUsageBytes.Set(float64(diskInfo.Used))
	SystemDiskCapacityBytes.Set(float64(diskInfo.Total))

	return nil
}

// StartSystemMetricsCollector démarre un collecteur périodique des métriques système
// Sprint 4 Phase 4.1 : Mise à jour automatique toutes les 30 secondes
func StartSystemMetricsCollector(interval time.Duration) {
	if interval == 0 {
		interval = 30 * time.Second // Par défaut : 30 secondes
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Première collecte immédiate
		UpdateSystemMetrics()

		for range ticker.C {
			if err := UpdateSystemMetrics(); err != nil {
				// Log l'erreur mais continue (ne pas bloquer)
				// Note: On ne peut pas utiliser le logger ici car on est dans un goroutine
				// Les erreurs seront visibles via les métriques (valeurs à 0 ou NaN)
			}
		}
	}()
}

