// Package store wraps DynamoDB access for journal entries.
//
// Single-table design:
//
//	PK = USER#<sub>           SK = ENTRY#<ulid>
//
// Item attrs: title (S, plaintext label), ciphertext (B), nonce (B),
// wrapped_dek (B), created_at (S, RFC3339).
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

var ErrNotFound = errors.New("entry not found")

type Entry struct {
	ID         string    `dynamodbav:"-"`
	Title      string    `dynamodbav:"title"`
	Ciphertext []byte    `dynamodbav:"ciphertext"`
	Nonce      []byte    `dynamodbav:"nonce"`
	WrappedDEK []byte    `dynamodbav:"wrapped_dek"`
	CreatedAt  time.Time `dynamodbav:"created_at"`
}

// DDBAPI is the subset of *dynamodb.Client the Store needs. Letting callers
// inject a mock keeps tests off the network.
type DDBAPI interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
	GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	DeleteItem(ctx context.Context, params *dynamodb.DeleteItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.DeleteItemOutput, error)
}

type Store struct {
	DDB   DDBAPI
	Table string
}

func userPK(userID string) string   { return "USER#" + userID }
func entrySK(entryID string) string { return "ENTRY#" + entryID }

func (s *Store) Put(ctx context.Context, userID, entryID string, e *Entry) error {
	item, err := attributevalue.MarshalMap(e)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	item["PK"] = &ddbtypes.AttributeValueMemberS{Value: userPK(userID)}
	item["SK"] = &ddbtypes.AttributeValueMemberS{Value: entrySK(entryID)}

	_, err = s.DDB.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.Table),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("ddb put: %w", err)
	}
	return nil
}

// EntryMeta is what list returns — no ciphertext, so the client can show
// a list without round-tripping to KMS for every row.
type EntryMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *Store) List(ctx context.Context, userID string) ([]EntryMeta, error) {
	out, err := s.DDB.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(s.Table),
		KeyConditionExpression: aws.String("PK = :pk AND begins_with(SK, :sk)"),
		ExpressionAttributeValues: map[string]ddbtypes.AttributeValue{
			":pk": &ddbtypes.AttributeValueMemberS{Value: userPK(userID)},
			":sk": &ddbtypes.AttributeValueMemberS{Value: "ENTRY#"},
		},
		ProjectionExpression: aws.String("SK, title, created_at"),
		ScanIndexForward:     aws.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("ddb query: %w", err)
	}

	metas := make([]EntryMeta, 0, len(out.Items))
	for _, item := range out.Items {
		sk, _ := item["SK"].(*ddbtypes.AttributeValueMemberS)
		var row struct {
			Title     string    `dynamodbav:"title"`
			CreatedAt time.Time `dynamodbav:"created_at"`
		}
		if err := attributevalue.UnmarshalMap(item, &row); err != nil {
			return nil, fmt.Errorf("unmarshal row: %w", err)
		}
		metas = append(metas, EntryMeta{
			ID:        sk.Value[len("ENTRY#"):],
			Title:     row.Title,
			CreatedAt: row.CreatedAt,
		})
	}
	return metas, nil
}

func (s *Store) Get(ctx context.Context, userID, entryID string) (*Entry, error) {
	out, err := s.DDB.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: userPK(userID)},
			"SK": &ddbtypes.AttributeValueMemberS{Value: entrySK(entryID)},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("ddb get: %w", err)
	}
	if out.Item == nil {
		return nil, ErrNotFound
	}
	var e Entry
	if err := attributevalue.UnmarshalMap(out.Item, &e); err != nil {
		return nil, fmt.Errorf("unmarshal entry: %w", err)
	}
	e.ID = entryID
	return &e, nil
}

func (s *Store) Delete(ctx context.Context, userID, entryID string) error {
	_, err := s.DDB.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(s.Table),
		Key: map[string]ddbtypes.AttributeValue{
			"PK": &ddbtypes.AttributeValueMemberS{Value: userPK(userID)},
			"SK": &ddbtypes.AttributeValueMemberS{Value: entrySK(entryID)},
		},
	})
	if err != nil {
		return fmt.Errorf("ddb delete: %w", err)
	}
	return nil
}
