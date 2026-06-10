package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/kms"

	"github.com/phekno/inkwell/api/internal/crypto"
	"github.com/phekno/inkwell/api/internal/store"
)

// fakeDDB implements store.DDBAPI for the update path.
type fakeDDB struct {
	puts      []dynamodb.PutItemInput
	updates   []dynamodb.UpdateItemInput
	updateErr error
	getResp   *dynamodb.GetItemOutput
}

func (f *fakeDDB) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.puts = append(f.puts, *in)
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDDB) Query(context.Context, *dynamodb.QueryInput, ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return &dynamodb.QueryOutput{}, nil
}

func (f *fakeDDB) GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &dynamodb.GetItemOutput{}, nil
}

func (f *fakeDDB) UpdateItem(_ context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updates = append(f.updates, *in)
	return &dynamodb.UpdateItemOutput{}, f.updateErr
}

func (f *fakeDDB) DeleteItem(context.Context, *dynamodb.DeleteItemInput, ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	return &dynamodb.DeleteItemOutput{}, nil
}

// fakeKMS implements crypto.KMSAPI. Seal needs a 32-byte data key.
type fakeKMS struct{ sealed bool }

func (f *fakeKMS) GenerateDataKey(_ context.Context, _ *kms.GenerateDataKeyInput, _ ...func(*kms.Options)) (*kms.GenerateDataKeyOutput, error) {
	f.sealed = true
	return &kms.GenerateDataKeyOutput{
		Plaintext:      make([]byte, 32),
		CiphertextBlob: []byte("wrapped"),
	}, nil
}

func (f *fakeKMS) Decrypt(_ context.Context, _ *kms.DecryptInput, _ ...func(*kms.Options)) (*kms.DecryptOutput, error) {
	return &kms.DecryptOutput{Plaintext: make([]byte, 32)}, nil
}

func newTestHandler(ddb *fakeDDB, k *fakeKMS) *handler {
	return &handler{
		log:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		store: &store.Store{DDB: ddb, Table: "entries"},
		seal:  &crypto.Sealer{KMS: k, KeyID: "key"},
	}
}

func patchReq(id, body string) events.APIGatewayV2HTTPRequest {
	req := events.APIGatewayV2HTTPRequest{Body: body}
	req.RequestContext.HTTP.Method = "PATCH"
	req.RequestContext.HTTP.Path = "/entries/" + id
	req.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
			Claims: map[string]string{"sub": "user-1"},
		},
	}
	return req
}

func postReq(body string) events.APIGatewayV2HTTPRequest {
	req := events.APIGatewayV2HTTPRequest{Body: body}
	req.RequestContext.HTTP.Method = "POST"
	req.RequestContext.HTTP.Path = "/entries"
	req.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
			Claims: map[string]string{"sub": "user-1"},
		},
	}
	return req
}

func getReq(id string) events.APIGatewayV2HTTPRequest {
	req := events.APIGatewayV2HTTPRequest{}
	req.RequestContext.HTTP.Method = "GET"
	req.RequestContext.HTTP.Path = "/entries/" + id
	req.RequestContext.Authorizer = &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
		JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
			Claims: map[string]string{"sub": "user-1"},
		},
	}
	return req
}

func TestGetReturnsFolderAndUpdatedAt(t *testing.T) {
	k := &fakeKMS{}
	seal := &crypto.Sealer{KMS: k, KeyID: "key"}
	env, err := seal.Seal(context.Background(), "user-1", []byte("hello"))
	if err != nil {
		t.Fatal(err)
	}

	ddb := &fakeDDB{getResp: &dynamodb.GetItemOutput{
		Item: map[string]ddbtypes.AttributeValue{
			"title":       &ddbtypes.AttributeValueMemberS{Value: "hi"},
			"folder":      &ddbtypes.AttributeValueMemberS{Value: "work/journal"},
			"ciphertext":  &ddbtypes.AttributeValueMemberB{Value: env.Ciphertext},
			"nonce":       &ddbtypes.AttributeValueMemberB{Value: env.Nonce},
			"wrapped_dek": &ddbtypes.AttributeValueMemberB{Value: env.WrappedDEK},
			"created_at":  &ddbtypes.AttributeValueMemberS{Value: "2026-01-01T00:00:00Z"},
			"updated_at":  &ddbtypes.AttributeValueMemberS{Value: "2026-02-02T00:00:00Z"},
		},
	}}
	h := newTestHandler(ddb, k)

	resp, err := h.handle(context.Background(), getReq("abc"))
	if err != nil {
		t.Fatal(err)
	}
	var v entryView
	if err := json.Unmarshal([]byte(resp.Body), &v); err != nil {
		t.Fatal(err)
	}
	if v.Folder != "work/journal" {
		t.Fatalf("folder = %q", v.Folder)
	}
	if v.Body != "hello" {
		t.Fatalf("body = %q", v.Body)
	}
	if v.UpdatedAt.IsZero() {
		t.Fatalf("updated_at should be populated")
	}
}

func TestCreateStoresNormalizedFolder(t *testing.T) {
	ddb := &fakeDDB{}
	h := newTestHandler(ddb, &fakeKMS{})

	resp, err := h.handle(context.Background(), postReq(`{"title":"hi","body":"x","folder":"/work/journal/"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d, body %s", resp.StatusCode, resp.Body)
	}
	if len(ddb.puts) != 1 {
		t.Fatalf("expected 1 put, got %d", len(ddb.puts))
	}
	folder := ddb.puts[0].Item["folder"].(*ddbtypes.AttributeValueMemberS).Value
	if folder != "work/journal" {
		t.Fatalf("stored folder = %q, want normalized work/journal", folder)
	}
	var v entryView
	if err := json.Unmarshal([]byte(resp.Body), &v); err != nil {
		t.Fatal(err)
	}
	if v.Folder != "work/journal" {
		t.Fatalf("view folder = %q", v.Folder)
	}
}

func TestNormalizeFolder(t *testing.T) {
	cases := map[string]string{
		"":                 "",
		"work":             "work",
		"/work/":           "work",
		"work//journal":    "work/journal",
		" work / journal ": "work/journal",
		"///":              "",
	}
	for in, want := range cases {
		if got := normalizeFolder(in); got != want {
			t.Errorf("normalizeFolder(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUpdateFolderOnlyDoesNotSeal(t *testing.T) {
	ddb := &fakeDDB{}
	k := &fakeKMS{}
	h := newTestHandler(ddb, k)

	resp, err := h.handle(context.Background(), patchReq("abc", `{"folder":"/work/journal/"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, body %s", resp.StatusCode, resp.Body)
	}
	if k.sealed {
		t.Fatalf("folder-only update must not call KMS")
	}
	if len(ddb.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(ddb.updates))
	}
	got := ddb.updates[0].ExpressionAttributeValues[":folder"].(*ddbtypes.AttributeValueMemberS).Value
	if got != "work/journal" {
		t.Fatalf("folder = %q, want normalized work/journal", got)
	}
}

func TestUpdateBodySeals(t *testing.T) {
	ddb := &fakeDDB{}
	k := &fakeKMS{}
	h := newTestHandler(ddb, k)

	resp, err := h.handle(context.Background(), patchReq("abc", `{"title":"hi","body":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if !k.sealed {
		t.Fatalf("body update must re-seal via KMS")
	}
}

func TestUpdateBlankTitleRejected(t *testing.T) {
	h := newTestHandler(&fakeDDB{}, &fakeKMS{})
	resp, _ := h.handle(context.Background(), patchReq("abc", `{"title":"   "}`))
	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUpdateUnknownIDReturns404(t *testing.T) {
	ddb := &fakeDDB{updateErr: &ddbtypes.ConditionalCheckFailedException{}}
	h := newTestHandler(ddb, &fakeKMS{})
	resp, _ := h.handle(context.Background(), patchReq("missing", `{"folder":"work"}`))
	if resp.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestUpdateReturnsView(t *testing.T) {
	h := newTestHandler(&fakeDDB{}, &fakeKMS{})
	resp, _ := h.handle(context.Background(), patchReq("abc", `{"folder":"work"}`))
	var v entryView
	if err := json.Unmarshal([]byte(resp.Body), &v); err != nil {
		t.Fatal(err)
	}
	if v.ID != "abc" || v.Folder != "work" {
		t.Fatalf("view = %+v", v)
	}
	_ = aws.String("") // keep import
}
