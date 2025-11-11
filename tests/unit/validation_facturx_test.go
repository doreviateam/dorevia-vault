package unit

import (
	"testing"
	"time"

	"github.com/doreviateam/dorevia-vault/internal/validation"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createValidFacturXXML crée un XML Factur-X valide pour les tests
func createValidFacturXXML() []byte {
	return []byte(`<?xml version="1.0" encoding="UTF-8"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
	<ID>F2025-00123</ID>
	<IssueDate>2025-01-15</IssueDate>
	<DueDate>2025-02-15</DueDate>
	<InvoiceTypeCode>380</InvoiceTypeCode>
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
	<AccountingCustomerParty>
		<Party>
			<PartyName>
				<Name>Client SA</Name>
			</PartyName>
			<PartyTaxScheme>
				<CompanyID>FR98765432109</CompanyID>
			</PartyTaxScheme>
		</Party>
	</AccountingCustomerParty>
	<TaxTotal>
		<TaxAmount Amount="31.67">31.67</TaxAmount>
	</TaxTotal>
	<LegalMonetaryTotal>
		<TaxExclusiveAmount>158.33</TaxExclusiveAmount>
		<TaxInclusiveAmount>190.00</TaxInclusiveAmount>
	</LegalMonetaryTotal>
	<InvoiceLine>
		<ID>1</ID>
		<InvoicedQuantity value="1">1</InvoicedQuantity>
		<LineExtensionAmount>158.33</LineExtensionAmount>
		<Item>
			<Description>Service de consultation</Description>
			<Price>
				<PriceAmount>158.33</PriceAmount>
			</Price>
		</Item>
		<TaxTotal>
			<TaxAmount Amount="31.67">31.67</TaxAmount>
			<TaxSubtotal>
				<TaxCategory>
					<Percent>20.0</Percent>
				</TaxCategory>
			</TaxSubtotal>
		</TaxTotal>
	</InvoiceLine>
</Invoice>`)
}

// TestNewFacturXValidator teste la création d'un validateur
func TestNewFacturXValidator(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	assert.NotNil(t, validator)
}

// TestFacturXValidator_Validate_ValidXML teste la validation d'un XML valide
func TestFacturXValidator_Validate_ValidXML(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	xmlContent := createValidFacturXXML()

	result, err := validator.Validate(xmlContent, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
	assert.NotNil(t, result.Metadata)
	assert.Equal(t, "F2025-00123", result.Metadata.InvoiceNumber)
	assert.Equal(t, "EUR", result.Metadata.Currency)
	assert.Equal(t, 158.33, result.Metadata.TotalHT)
	assert.Equal(t, 190.00, result.Metadata.TotalTTC)
	assert.Equal(t, 31.67, result.Metadata.TaxAmount)
	assert.Equal(t, "FR12345678901", result.Metadata.SellerVAT)
	assert.Equal(t, "FR98765432109", result.Metadata.BuyerVAT)
}

// TestFacturXValidator_Validate_InvalidXML teste la validation d'un XML invalide
func TestFacturXValidator_Validate_InvalidXML(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	invalidXML := []byte(`<?xml version="1.0"?><Invalid></Invalid>`)

	result, err := validator.Validate(invalidXML, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestFacturXValidator_Validate_MissingFields teste la validation avec champs manquants
func TestFacturXValidator_Validate_MissingFields(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	// XML sans ID ni date
	invalidXML := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
	<DocumentCurrencyCode>EUR</DocumentCurrencyCode>
</Invoice>`)

	result, err := validator.Validate(invalidXML, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestFacturXValidator_Validate_ExtractMetadata teste l'extraction de métadonnées
func TestFacturXValidator_Validate_ExtractMetadata(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	xmlContent := createValidFacturXXML()

	result, err := validator.Validate(xmlContent, "application/xml")
	require.NoError(t, err)
	require.NotNil(t, result.Metadata)

	// Vérifier les métadonnées extraites
	assert.Equal(t, "F2025-00123", result.Metadata.InvoiceNumber)
	assert.Equal(t, "EUR", result.Metadata.Currency)
	assert.Equal(t, "ACME Corp", result.Metadata.SellerName)
	assert.Equal(t, "Client SA", result.Metadata.BuyerName)
	
	// Vérifier la date
	expectedDate, _ := time.Parse("2006-01-02", "2025-01-15")
	assert.Equal(t, expectedDate, result.Metadata.InvoiceDate)
	
	// Vérifier la date d'échéance
	require.NotNil(t, result.Metadata.DueDate)
	expectedDueDate, _ := time.Parse("2006-01-02", "2025-02-15")
	assert.Equal(t, expectedDueDate, *result.Metadata.DueDate)
	
	// Vérifier les lignes de facture
	assert.Len(t, result.Metadata.LineItems, 1)
	if len(result.Metadata.LineItems) > 0 {
		item := result.Metadata.LineItems[0]
		assert.Equal(t, "Service de consultation", item.Description)
		assert.Equal(t, 1.0, item.Quantity)
		assert.Equal(t, 158.33, item.UnitPrice)
		assert.Equal(t, 20.0, item.TaxRate)
	}
}

// TestFacturXValidator_Validate_AmountMismatch teste la validation des montants
func TestFacturXValidator_Validate_AmountMismatch(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	// XML avec montants incohérents
	invalidXML := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
	<ID>F2025-00123</ID>
	<IssueDate>2025-01-15</IssueDate>
	<DocumentCurrencyCode>EUR</DocumentCurrencyCode>
	<AccountingSupplierParty>
		<Party>
			<PartyTaxScheme>
				<CompanyID>FR12345678901</CompanyID>
			</PartyTaxScheme>
		</Party>
	</AccountingSupplierParty>
	<TaxTotal>
		<TaxAmount Amount="10.00">10.00</TaxAmount>
	</TaxTotal>
	<LegalMonetaryTotal>
		<TaxExclusiveAmount>100.00</TaxExclusiveAmount>
		<TaxInclusiveAmount>200.00</TaxInclusiveAmount>
	</LegalMonetaryTotal>
</Invoice>`)

	result, err := validator.Validate(invalidXML, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Devrait avoir un warning sur les montants
	assert.NotEmpty(t, result.Warnings)
}

// TestFacturXValidator_Validate_PDFWithXML teste l'extraction XML depuis PDF
func TestFacturXValidator_Validate_PDFWithXML(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	
	// Créer un PDF factice avec XML embarqué
	pdfHeader := []byte("%PDF-1.4\n")
	xmlContent := createValidFacturXXML()
	pdfFooter := []byte("\n%%EOF")
	
	pdfWithXML := append(pdfHeader, xmlContent...)
	pdfWithXML = append(pdfWithXML, pdfFooter...)

	result, err := validator.Validate(pdfWithXML, "application/pdf")
	require.NoError(t, err)
	assert.NotNil(t, result)
	// Devrait réussir à extraire et valider
	assert.True(t, result.Valid)
	assert.NotNil(t, result.Metadata)
}

// TestFacturXValidator_Validate_PDFWithoutXML teste un PDF sans XML
func TestFacturXValidator_Validate_PDFWithoutXML(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	
	// PDF sans XML
	pdfContent := []byte("%PDF-1.4\nSome PDF content\n%%EOF")

	result, err := validator.Validate(pdfContent, "application/pdf")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestFacturXValidator_Validate_EmptyContent teste avec contenu vide
func TestFacturXValidator_Validate_EmptyContent(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())

	result, err := validator.Validate([]byte{}, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
}

// TestFacturXValidator_Validate_MultipleLineItems teste avec plusieurs lignes
func TestFacturXValidator_Validate_MultipleLineItems(t *testing.T) {
	validator := validation.NewFacturXValidator(zerolog.Nop())
	
	xmlWithMultipleLines := []byte(`<?xml version="1.0"?>
<Invoice xmlns="urn:oasis:names:specification:ubl:schema:xsd:Invoice-2">
	<ID>F2025-00123</ID>
	<IssueDate>2025-01-15</IssueDate>
	<DocumentCurrencyCode>EUR</DocumentCurrencyCode>
	<AccountingSupplierParty>
		<Party>
			<PartyTaxScheme>
				<CompanyID>FR12345678901</CompanyID>
			</PartyTaxScheme>
		</Party>
	</AccountingSupplierParty>
	<TaxTotal>
		<TaxAmount Amount="40.00">40.00</TaxAmount>
	</TaxTotal>
	<LegalMonetaryTotal>
		<TaxExclusiveAmount>200.00</TaxExclusiveAmount>
		<TaxInclusiveAmount>240.00</TaxInclusiveAmount>
	</LegalMonetaryTotal>
	<InvoiceLine>
		<ID>1</ID>
		<InvoicedQuantity value="1">1</InvoicedQuantity>
		<LineExtensionAmount>100.00</LineExtensionAmount>
		<Item>
			<Description>Item 1</Description>
			<Price>
				<PriceAmount>100.00</PriceAmount>
			</Price>
		</Item>
		<TaxTotal>
			<TaxAmount Amount="20.00">20.00</TaxAmount>
			<TaxSubtotal>
				<TaxCategory>
					<Percent>20.0</Percent>
				</TaxCategory>
			</TaxSubtotal>
		</TaxTotal>
	</InvoiceLine>
	<InvoiceLine>
		<ID>2</ID>
		<InvoicedQuantity value="1">1</InvoicedQuantity>
		<LineExtensionAmount>100.00</LineExtensionAmount>
		<Item>
			<Description>Item 2</Description>
			<Price>
				<PriceAmount>100.00</PriceAmount>
			</Price>
		</Item>
		<TaxTotal>
			<TaxAmount Amount="20.00">20.00</TaxAmount>
			<TaxSubtotal>
				<TaxCategory>
					<Percent>20.0</Percent>
				</TaxCategory>
			</TaxSubtotal>
		</TaxTotal>
	</InvoiceLine>
</Invoice>`)

	result, err := validator.Validate(xmlWithMultipleLines, "application/xml")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Valid)
	require.NotNil(t, result.Metadata)
	assert.Len(t, result.Metadata.LineItems, 2)
}

