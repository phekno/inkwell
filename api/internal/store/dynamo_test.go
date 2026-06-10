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
