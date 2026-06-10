package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/oklog/ulid/v2"

	"github.com/phekno/inkwell/api/internal/crypto"
	"github.com/phekno/inkwell/api/internal/httpx"
	"github.com/phekno/inkwell/api/internal/store"
)

type handler struct {
	log   *slog.Logger
	store *store.Store
	seal  *crypto.Sealer
}

func (h *handler) handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path
	h.log.InfoContext(ctx, "request",
		"method", method, "path", path,
		"request_id", req.RequestContext.RequestID)

	if method == "GET" && path == "/health" {
		return httpx.JSON(200, map[string]string{"status": "ok"})
	}

	userID, ok := req.RequestContext.Authorizer.JWT.Claims["sub"]
	if !ok || userID == "" {
		return httpx.JSON(401, map[string]string{"error": "missing sub"})
	}

	switch {
	case method == "GET" && path == "/entries":
		return h.list(ctx, userID)
	case method == "POST" && path == "/entries":
		return h.create(ctx, userID, req.Body)
	case method == "GET" && strings.HasPrefix(path, "/entries/"):
		return h.get(ctx, userID, strings.TrimPrefix(path, "/entries/"))
	case method == "PATCH" && strings.HasPrefix(path, "/entries/"):
		return h.update(ctx, userID, strings.TrimPrefix(path, "/entries/"), req.Body)
	case method == "DELETE" && strings.HasPrefix(path, "/entries/"):
		return h.delete(ctx, userID, strings.TrimPrefix(path, "/entries/"))
	}
	return httpx.JSON(404, map[string]string{"error": "not found"})
}

func (h *handler) list(ctx context.Context, userID string) (events.APIGatewayV2HTTPResponse, error) {
	metas, err := h.store.List(ctx, userID)
	if err != nil {
		h.log.ErrorContext(ctx, "list failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "list failed"})
	}
	return httpx.JSON(200, metas)
}

type createReq struct {
	Title  string `json:"title"`
	Body   string `json:"body"`
	Folder string `json:"folder"`
}

type entryView struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Folder    string    `json:"folder"`
	Body      string    `json:"body,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (h *handler) create(ctx context.Context, userID, body string) (events.APIGatewayV2HTTPResponse, error) {
	var in createReq
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		return httpx.JSON(400, map[string]string{"error": "invalid json"})
	}
	if strings.TrimSpace(in.Title) == "" {
		return httpx.JSON(400, map[string]string{"error": "title required"})
	}

	env, err := h.seal.Seal(ctx, userID, []byte(in.Body))
	if err != nil {
		h.log.ErrorContext(ctx, "seal failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "seal failed"})
	}

	id := ulid.Make().String()
	now := time.Now().UTC()
	folder := normalizeFolder(in.Folder)
	if err := h.store.Put(ctx, userID, id, &store.Entry{
		Title:      in.Title,
		Folder:     folder,
		Ciphertext: env.Ciphertext,
		Nonce:      env.Nonce,
		WrappedDEK: env.WrappedDEK,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		h.log.ErrorContext(ctx, "put failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "put failed"})
	}

	return httpx.JSON(201, entryView{ID: id, Title: in.Title, Folder: folder, CreatedAt: now, UpdatedAt: now})
}

func (h *handler) get(ctx context.Context, userID, entryID string) (events.APIGatewayV2HTTPResponse, error) {
	e, err := h.store.Get(ctx, userID, entryID)
	if errors.Is(err, store.ErrNotFound) {
		return httpx.JSON(404, map[string]string{"error": "not found"})
	}
	if err != nil {
		h.log.ErrorContext(ctx, "get failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "get failed"})
	}
	pt, err := h.seal.Open(ctx, userID, &crypto.Envelope{
		Ciphertext: e.Ciphertext,
		Nonce:      e.Nonce,
		WrappedDEK: e.WrappedDEK,
	})
	if err != nil {
		h.log.ErrorContext(ctx, "open failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "open failed"})
	}
	return httpx.JSON(200, entryView{
		ID: entryID, Title: e.Title, Folder: e.Folder, Body: string(pt),
		CreatedAt: e.CreatedAt, UpdatedAt: e.UpdatedAt,
	})
}

// normalizeFolder canonicalizes a slash-path: trims whitespace, drops empty
// segments (collapsing repeated/leading/trailing slashes), and trims each
// segment. "" means root.
func normalizeFolder(s string) string {
	var segs []string
	for part := range strings.SplitSeq(s, "/") {
		if p := strings.TrimSpace(part); p != "" {
			segs = append(segs, p)
		}
	}
	return strings.Join(segs, "/")
}

type updateReq struct {
	Title  *string `json:"title"`
	Body   *string `json:"body"`
	Folder *string `json:"folder"`
}

func (h *handler) update(ctx context.Context, userID, entryID, body string) (events.APIGatewayV2HTTPResponse, error) {
	var in updateReq
	if err := json.Unmarshal([]byte(body), &in); err != nil {
		return httpx.JSON(400, map[string]string{"error": "invalid json"})
	}

	patch := store.EntryPatch{UpdatedAt: time.Now().UTC()}
	view := entryView{ID: entryID, UpdatedAt: patch.UpdatedAt}

	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return httpx.JSON(400, map[string]string{"error": "title required"})
		}
		patch.Title = &title
		view.Title = title
	}
	if in.Folder != nil {
		folder := normalizeFolder(*in.Folder)
		patch.Folder = &folder
		view.Folder = folder
	}
	if in.Body != nil {
		env, err := h.seal.Seal(ctx, userID, []byte(*in.Body))
		if err != nil {
			h.log.ErrorContext(ctx, "seal failed", "err", err)
			return httpx.JSON(500, map[string]string{"error": "seal failed"})
		}
		patch.Ciphertext = env.Ciphertext
		patch.Nonce = env.Nonce
		patch.WrappedDEK = env.WrappedDEK
	}

	if err := h.store.Update(ctx, userID, entryID, patch); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.JSON(404, map[string]string{"error": "not found"})
		}
		h.log.ErrorContext(ctx, "update failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "update failed"})
	}
	return httpx.JSON(200, view)
}

func (h *handler) delete(ctx context.Context, userID, entryID string) (events.APIGatewayV2HTTPResponse, error) {
	if err := h.store.Delete(ctx, userID, entryID); err != nil {
		h.log.ErrorContext(ctx, "delete failed", "err", err)
		return httpx.JSON(500, map[string]string{"error": "delete failed"})
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: 204}, nil
}

func main() {
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, "aws config:", err)
		os.Exit(1)
	}
	table := os.Getenv("ENTRIES_TABLE")
	keyID := os.Getenv("KMS_KEY_ID")
	if table == "" || keyID == "" {
		fmt.Fprintln(os.Stderr, "ENTRIES_TABLE and KMS_KEY_ID required")
		os.Exit(1)
	}
	h := &handler{
		log:   slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		store: &store.Store{DDB: dynamodb.NewFromConfig(cfg), Table: table},
		seal:  &crypto.Sealer{KMS: kms.NewFromConfig(cfg), KeyID: keyID},
	}
	lambda.Start(h.handle)
}
