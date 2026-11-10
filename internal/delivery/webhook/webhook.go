// Package webhook is the delivery.Deliverer for webhook destinations: it
// POSTs each envelope as JSON to the destination's URL, signed when the
// destination's config asks for it.
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"

	"github.com/harvor-io/relay/internal/delivery"
	"github.com/harvor-io/relay/internal/models"
	"github.com/harvor-io/relay/pkg/encryptor"
	"github.com/harvor-io/relay/pkg/secrets"
)

// Deliverer delivers envelopes to webhook destinations.
type Deliverer struct {
	client    *http.Client
	secrets   secrets.Store
	encryptor encryptor.Encryptor
}

var _ delivery.Deliverer = (*Deliverer)(nil)

// New returns a Deliverer that sends requests with client, and reads each
// destination's signing secret from secretStore, decrypting it with enc.
// The client's timeout bounds each delivery.
func New(client *http.Client, secretStore secrets.Store, enc encryptor.Encryptor) *Deliverer {
	return &Deliverer{client: client, secrets: secretStore, encryptor: enc}
}

// message is the JSON body of every webhook request: the contract consumers
// depend on. It is an explicit projection of models.Envelope, kept apart
// from the eventbus's own wire format, so neither changes the other.
type message struct {
	ID        string          `json:"id"`
	Source    string          `json:"source"`
	Type      string          `json:"type"`
	Topic     string          `json:"topic"`
	Data      json.RawMessage `json:"data"`
	CreatedAt time.Time       `json:"created_at"`
}

// Deliver POSTs envelope to destination's URL as JSON, signing it when the
// destination's config has hmac auth. Any response other than 2xx is an
// error. No error it returns contains the URL, the secret, or the
// signature: a URL can carry a token in its query string.
func (d *Deliverer) Deliver(ctx context.Context, destination *models.Destination, envelope *models.Envelope) error {
	if destination.Type != models.DestinationTypeWebhook {
		return fmt.Errorf("webhook: destination %s has type %q", destination.ID, destination.Type)
	}

	var cfg models.WebhookConfig
	if err := json.Unmarshal(destination.Config, &cfg); err != nil {
		return fmt.Errorf("webhook: decode destination %s config: %w", destination.ID, err)
	}

	body, err := json.Marshal(message{
		ID:        envelope.ID.String(),
		Source:    envelope.Source,
		Type:      envelope.Type,
		Topic:     envelope.Topic().String(),
		Data:      envelope.Data,
		CreatedAt: envelope.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("webhook: marshal envelope %s: %w", envelope.ID, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		// The error names the URL, so it isn't wrapped.
		return fmt.Errorf("webhook: destination %s has an invalid url", destination.ID)
	}
	req.Header.Set("Content-Type", "application/json")

	if cfg.Auth != nil {
		if err := d.sign(ctx, req, cfg.Auth, body); err != nil {
			return fmt.Errorf("webhook: sign request to destination %s: %w", destination.ID, err)
		}
	}

	resp, err := d.client.Do(req)
	if err != nil {
		// *url.Error names the URL, so report only its cause.
		return fmt.Errorf("webhook: deliver to destination %s: %w", destination.ID, unwrapURLError(err))
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("webhook: destination %s responded %d", destination.ID, resp.StatusCode)
	}
	return nil
}

// sign sets the timestamp and signature headers auth asks for on req,
// whose body is body.
func (d *Deliverer) sign(ctx context.Context, req *http.Request, auth *models.AuthConfig, body []byte) error {
	if auth.Type != models.AuthTypeHMAC {
		return fmt.Errorf("unsupported auth type %q", auth.Type)
	}
	var cfg models.HMACAuthConfig
	if err := json.Unmarshal(auth.Config, &cfg); err != nil {
		return fmt.Errorf("decode hmac config: %w", err)
	}

	newHash, err := hashFor(cfg.Algorithm)
	if err != nil {
		return err
	}
	encode, err := encoderFor(cfg.Encoding)
	if err != nil {
		return err
	}

	key, err := d.secret(ctx, cfg.SecretID)
	if err != nil {
		return err
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signed := strings.NewReplacer(
		"{timestamp}", timestamp,
		"{body}", string(body),
	).Replace(cfg.SigningTemplate)

	mac := hmac.New(newHash, key)
	mac.Write([]byte(signed))

	req.Header.Set(cfg.TimestampHeader, timestamp)
	req.Header.Set(cfg.SignatureHeader, encode(mac.Sum(nil)))
	return nil
}

// secret returns the decrypted signing secret stored under id.
func (d *Deliverer) secret(ctx context.Context, id string) ([]byte, error) {
	secretID, err := uuid.FromString(id)
	if err != nil {
		return nil, fmt.Errorf("parse secret id: %w", err)
	}
	stored, err := d.secrets.Get(ctx, secretID)
	if err != nil {
		return nil, fmt.Errorf("get secret %s: %w", secretID, err)
	}
	key, err := d.encryptor.Decrypt(stored.Value)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret %s: %w", secretID, err)
	}
	return key, nil
}

// hashFor returns the hash constructor for an hmac config's algorithm.
func hashFor(algorithm string) (func() hash.Hash, error) {
	switch algorithm {
	case "sha256":
		return sha256.New, nil
	case "sha512":
		return sha512.New, nil
	default:
		return nil, fmt.Errorf("unsupported hmac algorithm %q", algorithm)
	}
}

// encoderFor returns the function that encodes signature bytes for an hmac
// config's encoding.
func encoderFor(encoding string) (func([]byte) string, error) {
	switch encoding {
	case "hex":
		return hex.EncodeToString, nil
	case "base64":
		return base64.StdEncoding.EncodeToString, nil
	default:
		return nil, fmt.Errorf("unsupported signature encoding %q", encoding)
	}
}

// unwrapURLError strips the *url.Error that http.Client.Do wraps every
// error in, since its message includes the request URL.
func unwrapURLError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return urlErr.Err
	}
	return err
}
