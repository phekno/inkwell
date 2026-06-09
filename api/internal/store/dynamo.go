// Package store wraps DynamoDB access for journal entries.
//
// Single-table design:
//
//	PK = USER#<sub>           SK = ENTRY#<ulid>
//
// Item attrs: ciphertext (B), nonce (B), wrapped_dek (B), kms_key_id (S),
// created_at (S, RFC3339), title (S, plaintext, user-supplied label).
package store

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Entry struct {
	UserID     string
	EntryID    string
	Title      string
	Ciphertext []byte
	Nonce      []byte
	WrappedDEK []byte
	KMSKeyID   string
	CreatedAt  string
}

type Store struct {
	DDB   *dynamodb.Client
	Table string
}

func (s *Store) Put(ctx context.Context, e *Entry) error {
	return errors.New("not implemented")
}

func (s *Store) List(ctx context.Context, userID string) ([]Entry, error) {
	return nil, errors.New("not implemented")
}

func (s *Store) Get(ctx context.Context, userID, entryID string) (*Entry, error) {
	return nil, errors.New("not implemented")
}
