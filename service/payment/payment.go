package payment

import (
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/paymentintent"
)

// Set the stripe secret key system-wide for the Stripe SDK to work
func InitStripe(key string) {
	stripe.Key = key
}

// Method to create payment intent, return the client secret of the intent, or error
func CreatePaymentIntent(amount int64) (string, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amount),
		Currency: stripe.String(string(stripe.CurrencyUSD)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}

	intent, err := paymentintent.New(params)
	if err != nil {
		return "", err
	}

	return intent.ClientSecret, nil
}
