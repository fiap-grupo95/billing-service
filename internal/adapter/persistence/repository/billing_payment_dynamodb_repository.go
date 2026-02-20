package repository

import (
	"context"
	"time"

	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase/interfaces"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	defaultPaymentsTableName = "payments"
	paymentsEstimateIDIndex  = "estimate_id-index"
)

type billingPaymentItem struct {
	ID           string                 `dynamodbav:"id"`
	EstimateID   string                 `dynamodbav:"estimate_id"`
	Date         string                 `dynamodbav:"date"`
	Status       string                 `dynamodbav:"status"`
	MPPayload    map[string]interface{} `dynamodbav:"mp_payload,omitempty"`
	MPPayloadRaw string                 `dynamodbav:"mp_payload_raw,omitempty"`
}

// BillingPaymentDynamoRepository persists BillingPayment entities in DynamoDB.
//
// Table requirements:
//   - PK: id (string)
//   - GSI: estimate_id-index (PK: estimate_id)

type BillingPaymentDynamoRepository struct {
	ddb       *dynamodb.Client
	tableName string
}

var _ interfaces.IBillingPaymentRepository = (*BillingPaymentDynamoRepository)(nil)

func NewBillingPaymentDynamoRepository(ddb *dynamodb.Client) *BillingPaymentDynamoRepository {
	return &BillingPaymentDynamoRepository{
		ddb:       ddb,
		tableName: getenvDefault("PAYMENTS_TABLE", defaultPaymentsTableName),
	}
}

func (r *BillingPaymentDynamoRepository) Create(ctx context.Context, p entities.BillingPayment) (entities.BillingPayment, error) {
	it := toBillingPaymentItem(p)
	av, err := attributevalue.MarshalMap(it)
	if err != nil {
		return entities.BillingPayment{}, err
	}

	_, err = r.ddb.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(r.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(#id)"),
		ExpressionAttributeNames: map[string]string{
			"#id": "id",
		},
	})
	if err != nil {
		return entities.BillingPayment{}, err
	}
	return p, nil
}

func (r *BillingPaymentDynamoRepository) GetByID(ctx context.Context, id string) (entities.BillingPayment, error) {
	out, err := r.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return entities.BillingPayment{}, err
	}
	if len(out.Item) == 0 {
		return entities.BillingPayment{}, nil
	}

	var it billingPaymentItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return entities.BillingPayment{}, err
	}
	return fromBillingPaymentItem(it), nil
}

func (r *BillingPaymentDynamoRepository) ListByEstimateID(ctx context.Context, estimateID string) ([]entities.BillingPayment, error) {
	out, err := r.ddb.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(r.tableName),
		IndexName:              aws.String(paymentsEstimateIDIndex),
		KeyConditionExpression: aws.String("estimate_id = :eid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":eid": &types.AttributeValueMemberS{Value: estimateID},
		},
	})
	if err != nil {
		return nil, err
	}

	items := make([]entities.BillingPayment, 0, len(out.Items))
	for _, raw := range out.Items {
		var it billingPaymentItem
		if err := attributevalue.UnmarshalMap(raw, &it); err != nil {
			return nil, err
		}
		items = append(items, fromBillingPaymentItem(it))
	}
	return items, nil
}

func toBillingPaymentItem(p entities.BillingPayment) billingPaymentItem {
	return billingPaymentItem{
		ID:           p.ID,
		EstimateID:   p.EstimateID,
		Date:         p.Date.UTC().Format(time.RFC3339Nano),
		Status:       string(p.Status),
		MPPayload:    p.MPPayload,
		MPPayloadRaw: string(p.MPPayloadRaw),
	}
}

func fromBillingPaymentItem(it billingPaymentItem) entities.BillingPayment {
	dt, _ := time.Parse(time.RFC3339Nano, it.Date)
	return entities.BillingPayment{
		ID:           it.ID,
		EstimateID:   it.EstimateID,
		Date:         dt,
		Status:       entities.PaymentStatus(it.Status),
		MPPayload:    it.MPPayload,
		MPPayloadRaw: []byte(it.MPPayloadRaw),
	}
}
