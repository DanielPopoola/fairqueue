package paystack

// These types are internal to the Paystack adapter.
// They map to Paystack's specific API format and are
// never exposed outside this package.
// The service layer uses gateway.InitializeRequest etc. instead.

// apiResponse is the envelope Paystack wraps all responses in.
type apiResponse struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
}

// initializeData is the data field from Paystack's initialize response.
type initializeData struct {
	AuthorizationURL string `json:"authorization_url"`
	Reference        string `json:"reference"`
}

// verifyData is the data field from Paystack's verify response.
type verifyData struct {
	Status          string `json:"status"`
	GatewayResponse string `json:"gateway_response"`
	Reference       string `json:"reference"`
}
