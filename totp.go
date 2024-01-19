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

// GenerateTOTP handle generate TOTP slack command HTTP request.
func GenerateTOTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Prepare client and repository.
	projectID, err := metadata.NewClient(http.DefaultClient).ProjectID()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Get project ID error")
		log.Printf("Get project ID: %s.", err)
		return
	}

	secretRepository, err := newSecretRepository(ctx, projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Initialize repository error")
		log.Printf("New secret repository: %s.", err)
		return
	}
	defer secretRepository.close()

	datastoreClient, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Initialize client error")
		log.Fatalf("New Datastore client: %s.", err)
		return
	}
	defer datastoreClient.Close()

	tuc := newTOTPUseCase(datastoreClient, secretRepository)
	totpToken, err := tuc.generateTOTP(ctx, r.Header, r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Generate error")
		log.Fatalf("Generate TOTP token: %s.", err)
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
	datastoreClient  *datastore.Client
	secretRepository *secretRepository
}

func newTOTPUseCase(datastoreClient *datastore.Client, secretRepository *secretRepository) *totpUseCase {
	return &totpUseCase{datastoreClient: datastoreClient, secretRepository: secretRepository}
}

func (c *totpUseCase) generateTOTP(ctx context.Context, header http.Header, body io.Reader) (string, error) {
	// Parse body.
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}

	signingSecret, err := c.secretRepository.get(ctx, os.Getenv("SLACK_SIGNING_SECRET_SECRET_ID"))
	if err != nil {
		return "", fmt.Errorf("get signing secret: %w", err)
	}

	sv, err := slack.NewSecretsVerifier(header, signingSecret)
	if err != nil {
		return "", fmt.Errorf("new secret verifier: %w", err)
	}

	if _, err := sv.Write(bodyBytes); err != nil {
		return "", fmt.Errorf("write body: %w", err)
	}

	if err := sv.Ensure(); err != nil {
		return "", fmt.Errorf("ensure secret: %w", err)
	}

	bodyStr := string(bodyBytes)
	v, err := url.ParseQuery(bodyStr)
	if err != nil {
		return "", fmt.Errorf("parse body: %w", err)
	}

	// Validation body.
	if v.Get("team_domain") == "" || v.Get("channel_name") == "" || v.Get("user_name") == "" {
		return "", fmt.Errorf("invalid request body: %s", bodyStr)
	}

	// Save generate log.
	k := datastore.IncompleteKey("totp_generate_log", nil)
	l := totpGenerateLog{
		TeamDomain:  v.Get("team_domain"),
		ChannelName: v.Get("channel_name"),
		UserName:    v.Get("user_name"),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	if _, err := c.datastoreClient.Put(ctx, k, &l); err != nil {
		return "", fmt.Errorf("put TOTP generate log to Datastore: %w", err)
	}

	secret, err := c.secretRepository.get(ctx, os.Getenv("TOTP_SECRET_SECRET_ID"))
	if err != nil {
		return "", fmt.Errorf("get secret: %w", err)
	}

	secretBytes, err := hex.DecodeString(secret)
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

func (r *secretRepository) get(ctx context.Context, secretName string) (string, error) {
	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", r.projectID, secretName),
	}

	resp, err := r.client.AccessSecretVersion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("get secret: %w", err)
	}

	return string(resp.Payload.Data), nil
}

func (r *secretRepository) close() {
	r.client.Close()
}
