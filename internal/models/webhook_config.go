package models

import "encoding/json"

// WebhookConfig is the Config for a Destination whose Type is
// DestinationTypeWebhook: where to deliver data, and how to authenticate the
// outbound request.
type WebhookConfig struct {
	// URL is the HTTP endpoint envelopes are delivered to.
	URL string `json:"url"`

	// Auth is the authentication applied to outbound requests. A nil Auth
	// means the webhook is called without authentication.
	Auth *AuthConfig `json:"auth,omitempty"`
}

// AuthType identifies the authentication scheme an AuthConfig configures.
type AuthType string

const (
	// AuthTypeHMAC signs each outbound request with an HMAC computed over a
	// signing template, using a secret held in the secrets store. It is
	// currently the only supported AuthType.
	AuthTypeHMAC AuthType = "hmac"

	// Other authentication schemes are not supported yet.
	// AuthTypeBearer AuthType = "bearer"
	// AuthTypeAPIKey AuthType = "api_key"
)

// AuthConfig is a discriminated union: Type says how Config is interpreted.
// Today Type is always AuthTypeHMAC and Config decodes as HMACAuthConfig.
type AuthConfig struct {
	Type   AuthType        `json:"type"`
	Config json.RawMessage `json:"config"`
}

// HMACAuthConfig is the AuthConfig.Config shape for AuthTypeHMAC.
type HMACAuthConfig struct {
	// SecretID references the HMAC secret held in the secrets store.
	SecretID string `json:"secret_id"`

	// Algorithm is the HMAC hash algorithm, e.g. "sha256".
	Algorithm string `json:"algorithm"`

	// SignatureHeader is the HTTP header the computed signature is sent in.
	SignatureHeader string `json:"signature_header"`

	// Encoding is how the signature bytes are encoded into the header, e.g.
	// "hex".
	Encoding string `json:"encoding"`

	// SigningTemplate describes what is signed, e.g. "{timestamp}.{body}".
	SigningTemplate string `json:"signing_template"`

	// TimestampHeader is the HTTP header the request timestamp is sent in,
	// referenced by SigningTemplate.
	TimestampHeader string `json:"timestamp_header"`
}

// Other AuthConfig.Config shapes are not supported yet.
//
// type BearerAuthConfig struct {
// 	SecretID string `json:"secret_id"`
// }
//
// type APIKeyAuthConfig struct {
// 	SecretID string `json:"secret_id"`
// 	Location string `json:"location"`
// 	Name     string `json:"name"`
// }
