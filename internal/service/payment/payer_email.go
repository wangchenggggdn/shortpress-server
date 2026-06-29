package payment

import (
	"strings"

	goStripe "github.com/stripe/stripe-go/v82"
)

// StripeCheckoutPayerEmail returns the email the customer entered on Stripe Checkout.
func StripeCheckoutPayerEmail(session *goStripe.CheckoutSession) string {
	if session == nil {
		return ""
	}
	if session.CustomerDetails != nil {
		if email := strings.TrimSpace(session.CustomerDetails.Email); email != "" {
			return email
		}
	}
	return strings.TrimSpace(session.CustomerEmail)
}

// StripeInvoicePayerEmail returns the customer email on a Stripe invoice.
func StripeInvoicePayerEmail(invoice *goStripe.Invoice) string {
	if invoice == nil {
		return ""
	}
	return strings.TrimSpace(invoice.CustomerEmail)
}

// PayPalPayerEmail extracts payer email from PayPal webhook resource payloads.
func PayPalPayerEmail(resource map[string]interface{}) string {
	if resource == nil {
		return ""
	}
	if email := payPalEmailFromPerson(resource["payer"]); email != "" {
		return email
	}
	if email := payPalEmailFromPerson(resource["subscriber"]); email != "" {
		return email
	}
	return ""
}

func payPalEmailFromPerson(v interface{}) string {
	person, ok := v.(map[string]interface{})
	if !ok {
		return ""
	}
	if email, ok := person["email_address"].(string); ok {
		return strings.TrimSpace(email)
	}
	return ""
}
