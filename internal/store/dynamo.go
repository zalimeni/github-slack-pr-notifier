package store

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const stateKey = "state"

type State struct {
	PK           string `dynamodbav:"pk"`
	LastModified string `dynamodbav:"last_modified"`
}

type DedupRecord struct {
	PK         string `dynamodbav:"pk"`
	SentAt     string `dynamodbav:"sent_at"`
	SentAtUnix int64  `dynamodbav:"sent_at_unix"`
	ExpiresAt  int64  `dynamodbav:"expires_at,omitempty"`
}

type Dynamo struct {
	tableName string
	client    *dynamodb.Client
}

func NewDynamo(tableName string) *Dynamo {
	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
	if err != nil {
		panic(fmt.Sprintf("load aws config: %v", err))
	}
	return &Dynamo{
		tableName: tableName,
		client:    dynamodb.NewFromConfig(cfg),
	}
}

func (d *Dynamo) LoadState(ctx context.Context) (State, error) {
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: stateKey},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return State{}, fmt.Errorf("get state item: %w", err)
	}
	if len(out.Item) == 0 {
		return State{}, nil
	}

	var state State
	if err := attributevalue.UnmarshalMap(out.Item, &state); err != nil {
		return State{}, fmt.Errorf("unmarshal state item: %w", err)
	}
	return state, nil
}

func (d *Dynamo) SaveState(ctx context.Context, state State) error {
	state.PK = stateKey
	item, err := attributevalue.MarshalMap(state)
	if err != nil {
		return fmt.Errorf("marshal state item: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put state item: %w", err)
	}
	return nil
}

func (d *Dynamo) Seen(ctx context.Context, key string) (bool, error) {
	record, found, err := d.getDedupRecord(ctx, dedupKey(key))
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if record.ExpiresAt > 0 && time.Now().Unix() > record.ExpiresAt {
		return false, nil
	}
	return true, nil
}

func (d *Dynamo) SeenWithin(ctx context.Context, key string, window time.Duration, updatedAt time.Time) (bool, error) {
	record, found, err := d.getDedupRecord(ctx, windowKey(key))
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if record.ExpiresAt > 0 && time.Now().Unix() > record.ExpiresAt {
		return false, nil
	}
	if record.SentAtUnix == 0 || updatedAt.IsZero() {
		return false, nil
	}
	lastSentAt := time.Unix(record.SentAtUnix, 0).UTC()
	if updatedAt.Before(lastSentAt) {
		return true, nil
	}
	return updatedAt.Sub(lastSentAt) < window, nil
}

func (d *Dynamo) Record(ctx context.Context, key string, sentAt time.Time, ttl time.Duration) error {
	return d.putDedupRecord(ctx, dedupKey(key), sentAt, ttl)
}

func (d *Dynamo) RecordWindow(ctx context.Context, key string, sentAt time.Time, ttl time.Duration) error {
	return d.putDedupRecord(ctx, windowKey(key), sentAt, ttl)
}

func (d *Dynamo) putDedupRecord(ctx context.Context, key string, sentAt time.Time, ttl time.Duration) error {
	record := DedupRecord{
		PK:         key,
		SentAt:     sentAt.UTC().Format(time.RFC3339),
		SentAtUnix: sentAt.UTC().Unix(),
		ExpiresAt:  sentAt.Add(ttl).Unix(),
	}
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return fmt.Errorf("marshal dedupe item: %w", err)
	}

	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("put dedupe item: %w", err)
	}
	return nil
}

func (d *Dynamo) getDedupRecord(ctx context.Context, pk string) (DedupRecord, bool, error) {
	out, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pk},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return DedupRecord{}, false, fmt.Errorf("get dedupe item: %w", err)
	}
	if len(out.Item) == 0 {
		return DedupRecord{}, false, nil
	}

	var record DedupRecord
	if err := attributevalue.UnmarshalMap(out.Item, &record); err != nil {
		return DedupRecord{}, false, fmt.Errorf("unmarshal dedupe item: %w", err)
	}
	return record, true, nil
}

func dedupKey(key string) string {
	return "dedupe#" + key
}

func windowKey(key string) string {
	return "window#" + key
}
