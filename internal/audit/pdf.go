package audit

import (
	"fmt"
	"image/png"
	"strings"

	"github.com/doreviateam/dorevia-vault/internal/crypto"
	"github.com/jung-kurt/gofpdf/v2"
	"github.com/rs/zerolog"
	"github.com/skip2/go-qrcode"
)

// PDFGenerator génère des rapports PDF
type PDFGenerator struct {
	jwsService *crypto.Service
	log        zerolog.Logger
}

// NewPDFGenerator crée un nouveau générateur PDF
func NewPDFGenerator(jwsService *crypto.Service, log zerolog.Logger) *PDFGenerator {
	return &PDFGenerator{
		jwsService: jwsService,
		log:        log,
	}
}

// Generate génère un PDF à partir d'un AuditReport
func (g *PDFGenerator) Generate(report *AuditReport, outputPath string) error {
	// Créer le PDF
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 20, 15) // gauche, haut, droite
	pdf.SetAutoPageBreak(true, 20) // marge bas

	// Ajouter toutes les pages
	if err := g.addCoverPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add cover page: %w", err)
	}

	if err := g.addSummaryPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add summary page: %w", err)
	}

	if err := g.addDocumentStatsPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add document stats page: %w", err)
	}

	if err := g.addErrorStatsPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add error stats page: %w", err)
	}

	if err := g.addPerformancePage(pdf, report); err != nil {
		return fmt.Errorf("failed to add performance page: %w", err)
	}

	if err := g.addLedgerPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add ledger page: %w", err)
	}

	if err := g.addSignaturesPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add signatures page: %w", err)
	}

	if err := g.addMetadataPage(pdf, report); err != nil {
		return fmt.Errorf("failed to add metadata page: %w", err)
	}

	// Sauvegarder le PDF
	return pdf.OutputFileAndClose(outputPath)
}

// addCoverPage ajoute la page de garde
func (g *PDFGenerator) addCoverPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()

	// Titre principal
	pdf.SetFont("Arial", "B", 24)
	pdf.SetTextColor(0, 102, 204) // Bleu Dorevia #0066CC
	pdf.CellFormat(0, 30, "Rapport d'Audit", "", 0, "C", false, 0, "")
	pdf.Ln(10)

	pdf.SetFont("Arial", "B", 18)
	pdf.SetTextColor(0, 0, 0)
	pdf.CellFormat(0, 20, "Dorevia Vault", "", 0, "C", false, 0, "")
	pdf.Ln(20)

	// Période
	pdf.SetFont("Arial", "", 14)
	pdf.SetTextColor(102, 102, 102) // Gris #666666
	pdf.CellFormat(0, 15, fmt.Sprintf("Période : %s", report.Period.Label), "", 0, "C", false, 0, "")
	pdf.Ln(10)

	pdf.CellFormat(0, 10, fmt.Sprintf("Du %s au %s", report.Period.StartDate, report.Period.EndDate), "", 0, "C", false, 0, "")
	pdf.Ln(30)

	// Date de génération
	pdf.SetFont("Arial", "", 10)
	pdf.CellFormat(0, 10, fmt.Sprintf("Généré le : %s", report.Metadata.GeneratedAt), "", 0, "C", false, 0, "")
	pdf.Ln(40)

	// QR Code du hash (si disponible)
	if report.Metadata.ReportHash != "" {
		if err := g.addQRCode(pdf, report.Metadata.ReportHash, 85, 120, 40); err != nil {
			g.log.Warn().Err(err).Msg("Failed to add QR code to cover page")
		} else {
			pdf.SetFont("Arial", "", 8)
			pdf.SetTextColor(102, 102, 102)
			pdf.SetY(165)
			pdf.CellFormat(0, 5, "Hash SHA256 du rapport", "", 0, "C", false, 0, "")
		}
	}

	return nil
}

// addSummaryPage ajoute la page résumé exécutif
func (g *PDFGenerator) addSummaryPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Résumé Exécutif")

	// Tableau récapitulatif
	headers := []string{"Indicateur", "Valeur"}
	rows := [][]string{
		{"Total documents", fmt.Sprintf("%d", report.Summary.TotalDocuments)},
		{"Total erreurs", fmt.Sprintf("%d", report.Summary.TotalErrors)},
		{"Taux d'erreur", fmt.Sprintf("%.2f%%", report.Summary.ErrorRate)},
		{"Taille moyenne document", formatBytes(report.Summary.AvgDocumentSize)},
		{"Taille totale stockage", formatBytes(report.Summary.TotalStorageSize)},
		{"Entrées ledger", fmt.Sprintf("%d", report.Summary.TotalLedgerEntries)},
		{"Réconciliations", fmt.Sprintf("%d", report.Summary.TotalReconciliations)},
		{"Intégrité ledger", formatBool(report.Ledger.ChainIntegrity)},
	}

	if err := g.addTable(pdf, headers, rows); err != nil {
		return err
	}

	return nil
}

// addDocumentStatsPage ajoute la page statistiques documents
func (g *PDFGenerator) addDocumentStatsPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Statistiques Documents")

	// Par statut
	if len(report.Documents.ByStatus) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "Par statut", "", 0, "L", false, 0, "")
		pdf.Ln(8)

		headers := []string{"Statut", "Nombre"}
		rows := [][]string{}
		for status, count := range report.Documents.ByStatus {
			rows = append(rows, []string{status, fmt.Sprintf("%d", count)})
		}
		if err := g.addTable(pdf, headers, rows); err != nil {
			return err
		}
		pdf.Ln(10)
	}

	// Par source
	if len(report.Documents.BySource) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "Par source", "", 0, "L", false, 0, "")
		pdf.Ln(8)

		headers := []string{"Source", "Nombre"}
		rows := [][]string{}
		for source, count := range report.Documents.BySource {
			rows = append(rows, []string{source, fmt.Sprintf("%d", count)})
		}
		if err := g.addTable(pdf, headers, rows); err != nil {
			return err
		}
		pdf.Ln(10)
	}

	// Distribution des tailles
	if report.Documents.SizeDistribution.Max > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "Distribution des tailles", "", 0, "L", false, 0, "")
		pdf.Ln(8)

		headers := []string{"Métrique", "Valeur"}
		rows := [][]string{
			{"Min", formatBytes(report.Documents.SizeDistribution.Min)},
			{"Max", formatBytes(report.Documents.SizeDistribution.Max)},
			{"Moyenne", formatBytes(int64(report.Documents.SizeDistribution.Mean))},
			{"Médiane", formatBytes(report.Documents.SizeDistribution.Median)},
			{"P95", formatBytes(report.Documents.SizeDistribution.P95)},
			{"P99", formatBytes(report.Documents.SizeDistribution.P99)},
		}
		if err := g.addTable(pdf, headers, rows); err != nil {
			return err
		}
	}

	return nil
}

// addErrorStatsPage ajoute la page statistiques erreurs
func (g *PDFGenerator) addErrorStatsPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Statistiques Erreurs")

	// Résumé
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 10, fmt.Sprintf("Total erreurs : %d", report.Errors.Total), "", 0, "L", false, 0, "")
	pdf.Ln(10)

	// Top 10 erreurs critiques
	if len(report.Errors.CriticalErrors) > 0 {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "Erreurs critiques (Top 10)", "", 0, "L", false, 0, "")
		pdf.Ln(8)

		headers := []string{"Type", "Document ID", "Occurrences", "Date"}
		rows := [][]string{}
		for _, err := range report.Errors.CriticalErrors {
			if len(rows) >= 10 {
				break
			}
			rows = append(rows, []string{
				err.EventType,
				truncateString(err.DocumentID, 20),
				fmt.Sprintf("%d", err.Count),
				formatDate(err.Timestamp),
			})
		}
		if err := g.addTable(pdf, headers, rows); err != nil {
			return err
		}
	}

	return nil
}

// addPerformancePage ajoute la page performance
func (g *PDFGenerator) addPerformancePage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Performance")

	// Tableau des durées
	headers := []string{"Opération", "Count", "Moyenne (s)", "P50 (s)", "P95 (s)", "P99 (s)", "Min (s)", "Max (s)"}
	rows := [][]string{
		{
			"Stockage documents",
			fmt.Sprintf("%d", report.Performance.DocumentStorage.Count),
			formatFloat(report.Performance.DocumentStorage.Mean),
			formatFloat(report.Performance.DocumentStorage.P50),
			formatFloat(report.Performance.DocumentStorage.P95),
			formatFloat(report.Performance.DocumentStorage.P99),
			formatFloat(report.Performance.DocumentStorage.Min),
			formatFloat(report.Performance.DocumentStorage.Max),
		},
		{
			"Signature JWS",
			fmt.Sprintf("%d", report.Performance.JWSSignature.Count),
			formatFloat(report.Performance.JWSSignature.Mean),
			formatFloat(report.Performance.JWSSignature.P50),
			formatFloat(report.Performance.JWSSignature.P95),
			formatFloat(report.Performance.JWSSignature.P99),
			formatFloat(report.Performance.JWSSignature.Min),
			formatFloat(report.Performance.JWSSignature.Max),
		},
		{
			"Ajout ledger",
			fmt.Sprintf("%d", report.Performance.LedgerAppend.Count),
			formatFloat(report.Performance.LedgerAppend.Mean),
			formatFloat(report.Performance.LedgerAppend.P50),
			formatFloat(report.Performance.LedgerAppend.P95),
			formatFloat(report.Performance.LedgerAppend.P99),
			formatFloat(report.Performance.LedgerAppend.Min),
			formatFloat(report.Performance.LedgerAppend.Max),
		},
		{
			"Transactions",
			fmt.Sprintf("%d", report.Performance.Transaction.Count),
			formatFloat(report.Performance.Transaction.Mean),
			formatFloat(report.Performance.Transaction.P50),
			formatFloat(report.Performance.Transaction.P95),
			formatFloat(report.Performance.Transaction.P99),
			formatFloat(report.Performance.Transaction.Min),
			formatFloat(report.Performance.Transaction.Max),
		},
	}

	return g.addTable(pdf, headers, rows)
}

// addLedgerPage ajoute la page ledger & réconciliation
func (g *PDFGenerator) addLedgerPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Ledger & Réconciliation")

	// Statistiques ledger
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 10, "Statistiques Ledger", "", 0, "L", false, 0, "")
	pdf.Ln(8)

	headers := []string{"Indicateur", "Valeur"}
	rows := [][]string{
		{"Total entrées", fmt.Sprintf("%d", report.Ledger.TotalEntries)},
		{"Nouvelles entrées (période)", fmt.Sprintf("%d", report.Ledger.NewEntries)},
		{"Erreurs", fmt.Sprintf("%d", report.Ledger.Errors)},
		{"Taux d'erreur", fmt.Sprintf("%.2f%%", report.Ledger.ErrorRate)},
		{"Taille actuelle", fmt.Sprintf("%d", report.Ledger.CurrentSize)},
		{"Intégrité chaîne", formatBool(report.Ledger.ChainIntegrity)},
		{"Dernier hash", truncateString(report.Ledger.LastHash, 40)},
	}
	if err := g.addTable(pdf, headers, rows); err != nil {
		return err
	}

	pdf.Ln(10)

	// Statistiques réconciliation
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 10, "Statistiques Réconciliation", "", 0, "L", false, 0, "")
	pdf.Ln(8)

	headers = []string{"Indicateur", "Valeur"}
	rows = [][]string{
		{"Total exécutions", fmt.Sprintf("%d", report.Reconciliation.TotalRuns)},
		{"Exécutions réussies", fmt.Sprintf("%d", report.Reconciliation.SuccessfulRuns)},
		{"Exécutions échouées", fmt.Sprintf("%d", report.Reconciliation.FailedRuns)},
		{"Fichiers orphelins trouvés", fmt.Sprintf("%d", report.Reconciliation.OrphanFilesFound)},
		{"Fichiers orphelins corrigés", fmt.Sprintf("%d", report.Reconciliation.OrphanFilesFixed)},
		{"Documents corrigés", fmt.Sprintf("%d", report.Reconciliation.DocumentsFixed)},
	}
	return g.addTable(pdf, headers, rows)
}

// addSignaturesPage ajoute la page signatures journalières
func (g *PDFGenerator) addSignaturesPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Signatures Journalières")

	if len(report.Signatures) == 0 {
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 10, "Aucune signature disponible pour cette période", "", 0, "C", false, 0, "")
		return nil
	}

	headers := []string{"Date", "Hash", "Lignes", "Timestamp"}
	rows := [][]string{}
	for _, sig := range report.Signatures {
		rows = append(rows, []string{
			sig.Date,
			truncateString(sig.Hash, 30),
			fmt.Sprintf("%d", sig.LineCount),
			formatDate(sig.Timestamp),
		})
	}

	return g.addTable(pdf, headers, rows)
}

// addMetadataPage ajoute la page métadonnées
func (g *PDFGenerator) addMetadataPage(pdf *gofpdf.Fpdf, report *AuditReport) error {
	pdf.AddPage()
	g.addPageHeader(pdf, "Métadonnées")

	// Informations système
	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 10, "Informations système", "", 0, "L", false, 0, "")
	pdf.Ln(8)

	headers := []string{"Champ", "Valeur"}
	rows := [][]string{
		{"Version", report.Metadata.Version},
		{"Généré par", report.Metadata.GeneratedBy},
		{"Généré le", formatDate(report.Metadata.GeneratedAt)},
		{"Report ID", report.Metadata.ReportID},
		{"Report Hash", truncateString(report.Metadata.ReportHash, 50)},
		{"Sources de données", strings.Join(report.Metadata.DataSources, ", ")},
	}
	if err := g.addTable(pdf, headers, rows); err != nil {
		return err
	}

	pdf.Ln(10)

	// Signature JWS (si disponible)
	if report.Metadata.ReportJWS != "" {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 10, "Signature JWS", "", 0, "L", false, 0, "")
		pdf.Ln(8)

		pdf.SetFont("Arial", "", 8)
		pdf.SetTextColor(102, 102, 102)
		pdf.MultiCell(0, 5, truncateString(report.Metadata.ReportJWS, 100), "", "", false)
	}

	return nil
}

// addPageHeader ajoute un en-tête de page
func (g *PDFGenerator) addPageHeader(pdf *gofpdf.Fpdf, title string) {
	pdf.SetFont("Arial", "B", 16)
	pdf.SetTextColor(0, 102, 204) // Bleu Dorevia
	pdf.CellFormat(0, 15, title, "", 0, "L", false, 0, "")
	pdf.Ln(10)
	pdf.SetDrawColor(0, 102, 204)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(5)
	pdf.SetTextColor(0, 0, 0)
}

// addTable ajoute un tableau formaté
func (g *PDFGenerator) addTable(pdf *gofpdf.Fpdf, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}

	// Calculer la largeur des colonnes
	colWidth := 180.0 / float64(len(headers))

	// En-tête
	pdf.SetFont("Arial", "B", 10)
	pdf.SetFillColor(0, 102, 204) // Bleu Dorevia
	pdf.SetTextColor(255, 255, 255)
	for _, header := range headers {
		pdf.CellFormat(colWidth, 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Lignes
	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(0, 0, 0)
	fill := false
	for _, row := range rows {
		for i, cell := range row {
			if i < len(headers) {
				pdf.SetFillColor(240, 240, 240)
				pdf.CellFormat(colWidth, 7, cell, "1", 0, "L", fill, 0, "")
			}
		}
		pdf.Ln(-1)
		fill = !fill
	}

	return nil
}

// addQRCode ajoute un QR code (hash SHA256 du rapport)
func (g *PDFGenerator) addQRCode(pdf *gofpdf.Fpdf, hash string, x, y, size float64) error {
	// Générer le QR code
	qrCode, err := qrcode.New(hash, qrcode.Medium)
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Convertir en image PNG (256x256 pixels)
	qrImage := qrCode.Image(256)

	// Convertir en bytes PNG
	var buf strings.Builder
	if err := png.Encode(&buf, qrImage); err != nil {
		return fmt.Errorf("failed to encode QR code: %w", err)
	}

	// Ajouter l'image au PDF
	imageName := fmt.Sprintf("QR_%s", hash[:8])
	pdf.RegisterImageOptionsReader(imageName, gofpdf.ImageOptions{ImageType: "PNG"}, strings.NewReader(buf.String()))

	// Placer l'image
	pdf.ImageOptions(imageName, x, y, size, size, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")

	return nil
}

// Helper functions

func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func formatBool(b bool) string {
	if b {
		return "✓ Oui"
	}
	return "✗ Non"
}

func formatFloat(f float64) string {
	if f == 0 {
		return "0.00"
	}
	return fmt.Sprintf("%.3f", f)
}

func formatDate(dateStr string) string {
	// Format RFC3339 -> DD/MM/YYYY HH:MM
	if len(dateStr) >= 19 {
		return dateStr[:10] + " " + dateStr[11:16]
	}
	return dateStr
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

