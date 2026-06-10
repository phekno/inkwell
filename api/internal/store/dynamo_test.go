package store

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type fakeDDB struct {
	puts        []dynamodb.PutItemInput
	queryResp   *dynamodb.QueryOutput
	getResp     *dynamodb.GetItemOutput
	getErr      error
	updates     []dynamodb.UpdateItemInput
	updateErr   error
	deleteCalls []dynamodb.DeleteItemInput
}

func (f *fakeDDB) PutItem(_ context.Context, in *dynamodb.PutItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	f.puts = append(f.puts, *in)
	return &dynamodb.PutItemOutput{}, nil
}

func (f *fakeDDB) Query(_ context.Context, _ *dynamodb.QueryInput, _ ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	return f.queryResp, nil
}

func (f *fakeDDB) GetItem(_ context.Context, _ *dynamodb.GetItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	return f.getResp, f.getErr
}

func (f *fakeDDB) UpdateItem(_ context.Context, in *dynamodb.UpdateItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	f.updates = append(f.updates, *in)
	return &dynamodb.UpdateItemOutput{}, f.updateErr
}

func (f *fakeDDB) DeleteItem(_ context.Context, in *dynamodb.DeleteItemInput, _ ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error) {
	f.deleteCalls = append(f.deleteCalls, *in)
	return &dynamodb.DeleteItemOutput{}, nil
}

func TestPutSetsPKAndSK(t *testing.T) {
	f := &fakeDDB{}
	s := &Store{DDB: f, Table: "entries"}

	err := s.Put(context.Background(), "user-1", "01HXYZ", &Entry{
		Title: "hi", Ciphertext: []byte{1}, Nonce: []byte{2}, WrappedDEK: []byte{3},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if len(f.puts) != 1 {
		t.Fatalf("expected 1 put, got %d", len(f.puts))
	}
	pk := f.puts[0].Item["PK"].(*ddbtypes.AttributeValueMemberS).Value
	sk := f.puts[0].Item["SK"].(*ddbtypes.AttributeValueMemberS).Value
	if pk != "USER#user-1" {
		t.Fatalf("PK = %q", pk)
	}
	if sk != "ENTRY#01HXYZ" {
		t.Fatalf("SK = %q", sk)
	}
}

func TestListExtractsEntryID(t *testing.T) {
	f := &fakeDDB{
		queryResp: &dynamodb.QueryOutput{
			Items: []map[string]ddbtypes.AttributeValue{
				{
					"SK":         &ddbtypes.AttributeValueMemberS{Value: "ENTRY#abc"},
					"title":      &ddbtypes.AttributeValueMemberS{Value: "first"},
					"created_at": &ddbtypes.AttributeValueMemberS{Value: "2026-01-01T00:00:00Z"},
				},
			},
		},
	}
	s := &Store{DDB: f, Table: "entries"}
	got, err := s.List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "abc" || got[0].Title != "first" {
		t.Fatalf("unexpected list: %+v", got)
	}
}

func TestListIncludesFolder(t *testing.T) {
	f := &fakeDDB{
		queryResp: &dynamodb.QueryOutput{
			Items: []map[string]ddbtypes.AttributeValue{
				{
					"SK":         &ddbtypes.AttributeValueMemberS{Value: "ENTRY#abc"},
					"title":      &ddbtypes.AttributeValueMemberS{Value: "first"},
					"folder":     &ddbtypes.AttributeValueMemberS{Value: "work/journal"},
					"created_at": &ddbtypes.AttributeValueMemberS{Value: "2026-01-01T00:00:00Z"},
				},
			},
		},
	}
	s := &Store{DDB: f, Table: "entries"}
	got, err := s.List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Folder != "work/journal" {
		t.Fatalf("Folder = %q, want work/journal", got[0].Folder)
	}
}

func TestListMissingFolderIsRoot(t *testing.T) {
	f := &fakeDDB{
		queryResp: &dynamodb.QueryOutput{
			Items: []map[string]ddbtypes.AttributeValue{
				{
					"SK":         &ddbtypes.AttributeValueMemberS{Value: "ENTRY#abc"},
					"title":      &ddbtypes.AttributeValueMemberS{Value: "legacy"},
					"created_at": &ddbtypes.AttributeValueMemberS{Value: "2026-01-01T00:00:00Z"},
				},
			},
		},
	}
	s := &Store{DDB: f, Table: "entries"}
	got, err := s.List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Folder != "" {
		t.Fatalf("Folder = %q, want empty (root)", got[0].Folder)
	}
}

func TestUpdateFolderOnly(t *testing.T) {
	f := &fakeDDB{}
	s := &Store{DDB: f, Table: "entries"}
	folder := "work/journal"
	err := s.Update(context.Background(), "user-1", "abc", EntryPatch{
		Folder:    &folder,
		UpdatedAt: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(f.updates) != 1 {
		t.Fatalf("expected 1 update, got %d", len(f.updates))
	}
	in := f.updates[0]
	if in.Key["PK"].(*ddbtypes.AttributeValueMemberS).Value != "USER#user-1" {
		t.Fatalf("bad PK")
	}
	if in.Key["SK"].(*ddbtypes.AttributeValueMemberS).Value != "ENTRY#abc" {
		t.Fatalf("bad SK")
	}
	if _, ok := in.ExpressionAttributeValues[":folder"]; !ok {
		t.Fatalf("missing :folder value")
	}
	if _, ok := in.ExpressionAttributeValues[":updated_at"]; !ok {
		t.Fatalf("updated_at should always be set")
	}
	if _, ok := in.ExpressionAttributeValues[":ciphertext"]; ok {
		t.Fatalf("folder-only update must not touch ciphertext")
	}
	if _, ok := in.ExpressionAttributeValues[":title"]; ok {
		t.Fatalf("folder-only update must not touch title")
	}
	if in.ConditionExpression == nil {
		t.Fatalf("expected an existence condition")
	}
}

func TestUpdateBodyResealsAllEnvelopeFields(t *testing.T) {
	f := &fakeDDB{}
	s := &Store{DDB: f, Table: "entries"}
	err := s.Update(context.Background(), "user-1", "abc", EntryPatch{
		Ciphertext: []byte{1},
		Nonce:      []byte{2},
		WrappedDEK: []byte{3},
		UpdatedAt:  time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	vals := f.updates[0].ExpressionAttributeValues
	for _, k := range []string{":ciphertext", ":nonce", ":wrapped_dek", ":updated_at"} {
		if _, ok := vals[k]; !ok {
			t.Fatalf("missing %s", k)
		}
	}
}

func TestUpdateMissingReturnsErrNotFound(t *testing.T) {
	f := &fakeDDB{updateErr: &ddbtypes.ConditionalCheckFailedException{}}
	s := &Store{DDB: f, Table: "entries"}
	title := "x"
	err := s.Update(context.Background(), "user-1", "missing", EntryPatch{
		Title:     &title,
		UpdatedAt: time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC),
	})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetMissingReturnsErrNotFound(t *testing.T) {
	f := &fakeDDB{getResp: &dynamodb.GetItemOutput{}} // nil Item
	s := &Store{DDB: f, Table: "entries"}
	_, err := s.Get(context.Background(), "user-1", "missing")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteUsesCompositeKey(t *testing.T) {
	f := &fakeDDB{}
	s := &Store{DDB: f, Table: "entries"}
	if err := s.Delete(context.Background(), "user-1", "abc"); err != nil {
		t.Fatal(err)
	}
	if f.deleteCalls[0].Key["PK"].(*ddbtypes.AttributeValueMemberS).Value != "USER#user-1" {
		t.Fatalf("bad PK")
	}
	if f.deleteCalls[0].Key["SK"].(*ddbtypes.AttributeValueMemberS).Value != "ENTRY#abc" {
		t.Fatalf("bad SK")
	}
	_ = aws.String("") // keep import used
}
