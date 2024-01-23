package totp

import (
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/hgfischer/go-otp"
	"github.com/slack-go/slack"
)

var tuc *totpUseCase

// Initialize datastore client and secrets.
func init() {
	ctx := context.Background()

	projectID, err := metadata.NewClient(http.DefaultClient).ProjectID()
	if err != nil {
		panic(err)
	}

	dc, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		panic(err)
	}

	secretRepository, err := newSecretRepository(ctx, projectID)
	if err != nil {
		panic(err)
	}

	ss, err := secretRepository.get(ctx, os.Getenv("SLACK_SIGNING_SECRET_SECRET_ID"))
	if err != nil {
		panic(err)
	}
	ts, err := secretRepository.get(ctx, os.Getenv("TOTP_SECRET_SECRET_ID"))
	if err != nil {
		panic(err)
	}

	tuc = &totpUseCase{
		datastoreClient: dc,
		signingSecret:   ss,
		totpSecret:      ts,
	}
}

// GenerateTOTP handle generate TOTP slack command HTTP request.
func GenerateTOTP(w http.ResponseWriter, r *http.Request) {
	totpToken, err := tuc.generateTOTP(r.Context(), r.Header, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Generate error")
		log.Printf("Generate totp token: %s.", err)
		return
	}

	fmt.Fprint(w, totpToken)
}

type totpGenerateLog struct {
	TeamDomain  string    `datastore:"team_domain"`
	ChannelName string    `datastore:"channel_name"`
	UserName    string    `datastore:"user_name"`
	CreatedAt   time.Time `datastore:"created_at"`
	UpdatedAt   time.Time `datastore:"updated_at"`
}

type totpUseCase struct {
	datastoreClient *datastore.Client

	signingSecret string
	totpSecret    string
}

func (c *totpUseCase) generateTOTP(ctx context.Context, header http.Header, body io.Reader) (string, error) {
	// Read body.
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	// Validating a request.
	sv, err := slack.NewSecretsVerifier(header, c.signingSecret)
	if err != nil {
		return "", fmt.Errorf("new secret verifier: %w", err)
	}
	if _, err := sv.Write(bodyBytes); err != nil {
		return "", fmt.Errorf("write body: %w", err)
	}
	if err := sv.Ensure(); err != nil {
		return "", fmt.Errorf("ensure secret: %w", err)
	}

	// Save generate log.
	v, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("parse body: %w", err)
	}
	k := datastore.IncompleteKey("totp_generate_log", nil)
	l := totpGenerateLog{
		TeamDomain:  v.Get("team_domain"),
		ChannelName: v.Get("channel_name"),
		UserName:    v.Get("user_name"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if _, err := c.datastoreClient.Put(ctx, k, &l); err != nil {
		return "", fmt.Errorf("put totp generate log to datastore: %w", err)
	}

	// Generate totp.
	secretBytes, err := hex.DecodeString(c.totpSecret)
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	totp := otp.TOTP{Secret: string(secretBytes), Time: time.Now(), Period: 60}
	return totp.Get(), nil
}

type secretRepository struct {
	client *secretmanager.Client

	projectID string
}

func newSecretRepository(ctx context.Context, projectID string) (*secretRepository, error) {
	c, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("new secret manager client: %w", err)
	}

	return &secretRepository{client: c, projectID: projectID}, nil
}

func (r *secretRepository) get(ctx context.Context, secretID string) (string, error) {
	resp, err := r.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", r.projectID, secretID),
	})
	if err != nil {
		return "", fmt.Errorf("get secret: %w", err)
	}

	return string(resp.Payload.Data), nil
}
