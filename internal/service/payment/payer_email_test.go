package payment

import (
	"testing"

	goStripe "github.com/stripe/stripe-go/v82"
)

func TestStripeCheckoutPayerEmail(t *testing.T) {
	session := &goStripe.CheckoutSession{
		CustomerDetails: &goStripe.CheckoutSessionCustomerDetails{
			Email: " checkout@example.com ",
		},
		CustomerEmail: "fallback@example.com",
	}
	if got := StripeCheckoutPayerEmail(session); got != "checkout@example.com" {
		t.Fatalf("expected trimmed customer details email, got %q", got)
	}

	session.CustomerDetails = nil
	if got := StripeCheckoutPayerEmail(session); got != "fallback@example.com" {
		t.Fatalf("expected customer email fallback, got %q", got)
	}
}

func TestPayPalPayerEmail(t *testing.T) {
	resource := map[string]interface{}{
		"payer": map[string]interface{}{
			"email_address": " payer@example.com ",
		},
	}
	if got := PayPalPayerEmail(resource); got != "payer@example.com" {
		t.Fatalf("expected payer email, got %q", got)
	}

	resource = map[string]interface{}{
		"subscriber": map[string]interface{}{
			"email_address": "sub@example.com",
		},
	}
	if got := PayPalPayerEmail(resource); got != "sub@example.com" {
		t.Fatalf("expected subscriber email, got %q", got)
	}
}
