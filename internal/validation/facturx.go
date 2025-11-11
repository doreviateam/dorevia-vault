package validation

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// FacturXValidator valide les factures Factur-X selon EN 16931
type FacturXValidator struct {
	log zerolog.Logger
}

// NewFacturXValidator crée un nouveau validateur Factur-X
func NewFacturXValidator(log zerolog.Logger) *FacturXValidator {
	return &FacturXValidator{
		log: log,
	}
}

// ValidationResult représente le résultat de la validation
type ValidationResult struct {
	Valid      bool     `json:"valid"`
	Errors     []string `json:"errors,omitempty"`
	Warnings   []string `json:"warnings,omitempty"`
	Metadata   *InvoiceMetadata `json:"metadata,omitempty"`
}

// InvoiceMetadata contient les métadonnées extraites de la facture
type InvoiceMetadata struct {
	InvoiceNumber string    `json:"invoice_number,omitempty"`
	InvoiceDate   time.Time `json:"invoice_date,omitempty"`
	DueDate       *time.Time `json:"due_date,omitempty"`
	TotalHT       float64   `json:"total_ht"`
	TotalTTC      float64   `json:"total_ttc"`
	Currency      string    `json:"currency,omitempty"`
	TaxAmount     float64   `json:"tax_amount"`
	SellerVAT     string    `json:"seller_vat,omitempty"`
	BuyerVAT      string    `json:"buyer_vat,omitempty"`
	SellerName    string    `json:"seller_name,omitempty"`
	BuyerName     string    `json:"buyer_name,omitempty"`
	LineItems     []LineItem `json:"line_items,omitempty"`
}

// LineItem représente une ligne de facture
type LineItem struct {
	Description string  `json:"description,omitempty"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TaxRate     float64 `json:"tax_rate"`
	TotalHT     float64 `json:"total_ht"`
	TotalTTC    float64 `json:"total_ttc"`
}

// Validate valide un document Factur-X
// Accepte soit un PDF avec XML embarqué, soit un XML pur
func (v *FacturXValidator) Validate(documentContent []byte, contentType string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Détecter le type de contenu
	if strings.Contains(contentType, "pdf") || strings.HasPrefix(string(documentContent), "%PDF") {
		// Extraire le XML depuis le PDF
		xmlContent, err := v.extractXMLFromPDF(documentContent)
		if err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to extract XML from PDF: %v", err))
			return result, nil
		}
		documentContent = xmlContent
	}

	// Parser le XML Factur-X
	invoice, err := v.parseFacturXXML(documentContent)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to parse XML: %v", err))
		return result, nil
	}

	// Valider la structure XML (validation basique)
	if err := v.validateXMLStructure(invoice); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("XML structure validation failed: %v", err))
		return result, nil
	}

	// Extraire les métadonnées
	metadata, err := v.extractMetadata(invoice)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to extract some metadata: %v", err))
	} else {
		result.Metadata = metadata
	}

	// Validation des champs obligatoires EN 16931
	if err := v.validateRequiredFields(metadata); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
	}

	// Validation des montants
	if err := v.validateAmounts(metadata); err != nil {
		result.Warnings = append(result.Warnings, err.Error())
	}

	return result, nil
}

// FacturXInvoice représente la structure XML Factur-X simplifiée
type FacturXInvoice struct {
	XMLName xml.Name `xml:"urn:oasis:names:specification:ubl:schema:xsd:Invoice-2 Invoice"`
	ID      string   `xml:"ID"`
	IssueDate string `xml:"IssueDate"`
	DueDate   string `xml:"DueDate,omitempty"`
	InvoiceTypeCode string `xml:"InvoiceTypeCode"`
	DocumentCurrencyCode string `xml:"DocumentCurrencyCode"`
	AccountingSupplierParty struct {
		Party struct {
			PartyName struct {
				Name string `xml:"Name"`
			} `xml:"PartyName"`
			PartyTaxScheme struct {
				CompanyID string `xml:"CompanyID"`
			} `xml:"PartyTaxScheme"`
		} `xml:"Party"`
	} `xml:"AccountingSupplierParty"`
	AccountingCustomerParty struct {
		Party struct {
			PartyName struct {
				Name string `xml:"Name"`
			} `xml:"PartyName"`
			PartyTaxScheme struct {
				CompanyID string `xml:"CompanyID"`
			} `xml:"PartyTaxScheme"`
		} `xml:"Party"`
	} `xml:"AccountingCustomerParty"`
	TaxTotal struct {
		TaxAmount struct {
			Amount string `xml:"Amount,attr"`
		} `xml:"TaxAmount"`
	} `xml:"TaxTotal"`
	LegalMonetaryTotal struct {
		TaxExclusiveAmount string `xml:"TaxExclusiveAmount"`
		TaxInclusiveAmount string `xml:"TaxInclusiveAmount"`
	} `xml:"LegalMonetaryTotal"`
	InvoiceLine []struct {
		ID          string `xml:"ID"`
		InvoicedQuantity struct {
			Value string `xml:"value,attr"`
		} `xml:"InvoicedQuantity"`
		LineExtensionAmount string `xml:"LineExtensionAmount"`
		Item struct {
			Description string `xml:"Description"`
			Price struct {
				PriceAmount string `xml:"PriceAmount"`
			} `xml:"Price"`
		} `xml:"Item"`
		TaxTotal struct {
			TaxAmount struct {
				Amount string `xml:"Amount,attr"`
			} `xml:"TaxAmount"`
			TaxSubtotal struct {
				TaxCategory struct {
					Percent string `xml:"Percent"`
				} `xml:"TaxCategory"`
			} `xml:"TaxSubtotal"`
		} `xml:"TaxTotal"`
	} `xml:"InvoiceLine"`
}

// extractXMLFromPDF extrait le XML Factur-X depuis un PDF
func (v *FacturXValidator) extractXMLFromPDF(pdfContent []byte) ([]byte, error) {
	// Rechercher le marqueur XML dans le PDF
	// Format Factur-X : le XML est embarqué dans le PDF avec le marqueur "<?xml"
	xmlStart := []byte("<?xml")
	xmlEnd := []byte("</Invoice>")
	
	startIdx := -1
	for i := 0; i < len(pdfContent)-len(xmlStart); i++ {
		if string(pdfContent[i:i+len(xmlStart)]) == string(xmlStart) {
			startIdx = i
			break
		}
	}
	
	if startIdx == -1 {
		return nil, fmt.Errorf("XML not found in PDF")
	}
	
	// Trouver la fin du XML
	endIdx := -1
	for i := startIdx; i < len(pdfContent)-len(xmlEnd); i++ {
		if string(pdfContent[i:i+len(xmlEnd)]) == string(xmlEnd) {
			endIdx = i + len(xmlEnd)
			break
		}
	}
	
	if endIdx == -1 {
		return nil, fmt.Errorf("XML end marker not found in PDF")
	}
	
	return pdfContent[startIdx:endIdx], nil
}

// parseFacturXXML parse le XML Factur-X
func (v *FacturXValidator) parseFacturXXML(xmlContent []byte) (*FacturXInvoice, error) {
	var invoice FacturXInvoice
	if err := xml.Unmarshal(xmlContent, &invoice); err != nil {
		return nil, fmt.Errorf("failed to unmarshal XML: %w", err)
	}
	return &invoice, nil
}

// validateXMLStructure valide la structure XML de base
func (v *FacturXValidator) validateXMLStructure(invoice *FacturXInvoice) error {
	if invoice.ID == "" {
		return fmt.Errorf("invoice ID is required")
	}
	if invoice.IssueDate == "" {
		return fmt.Errorf("invoice issue date is required")
	}
	if invoice.DocumentCurrencyCode == "" {
		return fmt.Errorf("document currency code is required")
	}
	return nil
}

// extractMetadata extrait les métadonnées de la facture
func (v *FacturXValidator) extractMetadata(invoice *FacturXInvoice) (*InvoiceMetadata, error) {
	metadata := &InvoiceMetadata{
		InvoiceNumber: invoice.ID,
		Currency:      invoice.DocumentCurrencyCode,
		SellerName:    invoice.AccountingSupplierParty.Party.PartyName.Name,
		BuyerName:     invoice.AccountingCustomerParty.Party.PartyName.Name,
	}

	// Parser la date
	if invoice.IssueDate != "" {
		date, err := time.Parse("2006-01-02", invoice.IssueDate)
		if err == nil {
			metadata.InvoiceDate = date
		}
	}

	// Parser la date d'échéance
	if invoice.DueDate != "" {
		date, err := time.Parse("2006-01-02", invoice.DueDate)
		if err == nil {
			metadata.DueDate = &date
		}
	}

	// Extraire les montants
	if invoice.LegalMonetaryTotal.TaxExclusiveAmount != "" {
		fmt.Sscanf(invoice.LegalMonetaryTotal.TaxExclusiveAmount, "%f", &metadata.TotalHT)
	}
	if invoice.LegalMonetaryTotal.TaxInclusiveAmount != "" {
		fmt.Sscanf(invoice.LegalMonetaryTotal.TaxInclusiveAmount, "%f", &metadata.TotalTTC)
	}
	if invoice.TaxTotal.TaxAmount.Amount != "" {
		fmt.Sscanf(invoice.TaxTotal.TaxAmount.Amount, "%f", &metadata.TaxAmount)
	}

	// Extraire les numéros de TVA
	if invoice.AccountingSupplierParty.Party.PartyTaxScheme.CompanyID != "" {
		metadata.SellerVAT = invoice.AccountingSupplierParty.Party.PartyTaxScheme.CompanyID
	}
	if invoice.AccountingCustomerParty.Party.PartyTaxScheme.CompanyID != "" {
		metadata.BuyerVAT = invoice.AccountingCustomerParty.Party.PartyTaxScheme.CompanyID
	}

	// Extraire les lignes de facture
	metadata.LineItems = make([]LineItem, 0, len(invoice.InvoiceLine))
	for _, line := range invoice.InvoiceLine {
		item := LineItem{
			Description: line.Item.Description,
		}
		
		if line.InvoicedQuantity.Value != "" {
			fmt.Sscanf(line.InvoicedQuantity.Value, "%f", &item.Quantity)
		}
		if line.Item.Price.PriceAmount != "" {
			fmt.Sscanf(line.Item.Price.PriceAmount, "%f", &item.UnitPrice)
		}
		if line.LineExtensionAmount != "" {
			fmt.Sscanf(line.LineExtensionAmount, "%f", &item.TotalHT)
		}
		if line.TaxTotal.TaxSubtotal.TaxCategory.Percent != "" {
			fmt.Sscanf(line.TaxTotal.TaxSubtotal.TaxCategory.Percent, "%f", &item.TaxRate)
		}
		if line.TaxTotal.TaxAmount.Amount != "" {
			var taxAmount float64
			fmt.Sscanf(line.TaxTotal.TaxAmount.Amount, "%f", &taxAmount)
			item.TotalTTC = item.TotalHT + taxAmount
		}
		
		metadata.LineItems = append(metadata.LineItems, item)
	}

	return metadata, nil
}

// validateRequiredFields valide les champs obligatoires EN 16931
func (v *FacturXValidator) validateRequiredFields(metadata *InvoiceMetadata) error {
	var errors []string

	if metadata.InvoiceNumber == "" {
		errors = append(errors, "invoice number is required")
	}
	if metadata.InvoiceDate.IsZero() {
		errors = append(errors, "invoice date is required")
	}
	if metadata.Currency == "" {
		errors = append(errors, "currency is required")
	}
	if metadata.SellerVAT == "" {
		errors = append(errors, "seller VAT number is required")
	}

	if len(errors) > 0 {
		return fmt.Errorf("required fields missing: %s", strings.Join(errors, ", "))
	}

	return nil
}

// validateAmounts valide la cohérence des montants
func (v *FacturXValidator) validateAmounts(metadata *InvoiceMetadata) error {
	// Vérifier que TotalTTC = TotalHT + TaxAmount (avec tolérance)
	expectedTTC := metadata.TotalHT + metadata.TaxAmount
	tolerance := 0.01 // Tolérance de 1 centime
	
	if metadata.TotalTTC > 0 && expectedTTC > 0 {
		diff := metadata.TotalTTC - expectedTTC
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			return fmt.Errorf("amount mismatch: TotalTTC (%.2f) != TotalHT (%.2f) + TaxAmount (%.2f)", 
				metadata.TotalTTC, metadata.TotalHT, metadata.TaxAmount)
		}
	}

	return nil
}

