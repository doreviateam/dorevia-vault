package services

// PosTicketInput représente l'input pour l'ingestion d'un ticket POS
// Type défini dans services pour éviter la dépendance inverse (services → handlers)
// Sprint 6 - Phase 0
type PosTicketInput struct {
	Tenant       string                 // Obligatoire
	SourceSystem string                 // Défaut: "odoo_pos"
	SourceModel  string                 // Obligatoire (ex: "pos.order")
	SourceID     string                 // Obligatoire
	Currency     *string                // Optionnel
	TotalInclTax *float64               // Optionnel
	TotalExclTax *float64               // Optionnel
	PosSession   *string                // Optionnel
	Cashier      *string                // Optionnel
	Location     *string                // Optionnel
	Ticket       map[string]interface{} // Obligatoire (JSON brut du ticket)
}

