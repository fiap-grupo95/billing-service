package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"mecanica_xpto/internal/domain/entities"
	"mecanica_xpto/internal/usecase/interfaces"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const defaultEstimatesTableName = "estimates"

type estimateItem struct {
	ID        string `dynamodbav:"id"`
	OSID      string `dynamodbav:"os_id"`
	Price     string `dynamodbav:"price"`
	Status    string `dynamodbav:"status"`
	CreatedAt string `dynamodbav:"created_at"`
	UpdatedAt string `dynamodbav:"updated_at"`
}

// EstimateDynamoRepository persists Estimate entities in DynamoDB.
//
// Table requirements:
//   - PK: id (string)
//
// We purposely use OS id as PK (estimate ID) to guarantee 1 estimate per OS.
// This keeps "PATCH /os/{id}/estimate" operations simple and efficient.

type EstimateDynamoRepository struct {
	ddb       *dynamodb.Client
	tableName string
}

var _ interfaces.IEstimateRepository = (*EstimateDynamoRepository)(nil)

func NewEstimateDynamoRepository(ddb *dynamodb.Client) *EstimateDynamoRepository {
	return &EstimateDynamoRepository{
		ddb:       ddb,
		tableName: getenvDefault("ESTIMATES_TABLE", defaultEstimatesTableName),
	}
}

func (r *EstimateDynamoRepository) Create(ctx context.Context, e entities.Estimate) (entities.Estimate, error) {
	it := toEstimateItem(e)
	av, err := attributevalue.MarshalMap(it)
	if err != nil {
		return entities.Estimate{}, err
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
		return entities.Estimate{}, err
	}
	return e, nil
}

func (r *EstimateDynamoRepository) GetByID(ctx context.Context, id string) (entities.Estimate, error) {
	out, err := r.ddb.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConsistentRead: aws.Bool(true),
	})
	if err != nil {
		return entities.Estimate{}, err
	}
	if len(out.Item) == 0 {
		return entities.Estimate{}, nil
	}

	var it estimateItem
	if err := attributevalue.UnmarshalMap(out.Item, &it); err != nil {
		return entities.Estimate{}, err
	}
	return fromEstimateItem(it), nil
}

func (r *EstimateDynamoRepository) GetByOSID(ctx context.Context, osID string) (entities.Estimate, error) {
	// Domain rule: estimate ID equals OS ID. We can resolve by PK directly.
	return r.GetByID(ctx, osID)
}

func (r *EstimateDynamoRepository) UpdateStatusByOSID(ctx context.Context, osID string, status entities.EstimateStatus) (entities.Estimate, error) {
	estimate, err := r.GetByOSID(ctx, osID)
	if err != nil {
		return entities.Estimate{}, err
	}
	if estimate.ID == "" {
		return entities.Estimate{}, nil
	}

	return r.update(ctx, estimate.ID, func(now string) (string, map[string]types.AttributeValue, map[string]string) {
		expr := "SET #status = :status, #updated_at = :updated_at"
		vals := map[string]types.AttributeValue{
			":status":     &types.AttributeValueMemberS{Value: string(status)},
			":updated_at": &types.AttributeValueMemberS{Value: now},
		}
		names := map[string]string{
			"#status":     "status",
			"#updated_at": "updated_at",
		}
		return expr, vals, names
	})
}

func (r *EstimateDynamoRepository) UpdatePriceByID(ctx context.Context, id string, newPrice float64) (entities.Estimate, error) {
	return r.update(ctx, id, func(now string) (string, map[string]types.AttributeValue, map[string]string) {
		expr := "SET #price = :price, #updated_at = :updated_at"
		vals := map[string]types.AttributeValue{
			":price":      &types.AttributeValueMemberN{Value: floatToString(newPrice)},
			":updated_at": &types.AttributeValueMemberS{Value: now},
		}
		names := map[string]string{
			"#price":      "price",
			"#updated_at": "updated_at",
		}
		return expr, vals, names
	})
}

func (r *EstimateDynamoRepository) update(
	ctx context.Context,
	id string,
	build func(now string) (updateExpr string, values map[string]types.AttributeValue, names map[string]string),
) (entities.Estimate, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	updateExpr, values, names := build(now)

	out, err := r.ddb.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableName),
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConditionExpression:       aws.String("attribute_exists(#id)"),
		UpdateExpression:          aws.String(updateExpr),
		ExpressionAttributeValues: values,
		ExpressionAttributeNames:  mergeNames(names, map[string]string{"#id": "id"}),
		ReturnValues:              types.ReturnValueAllNew,
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return entities.Estimate{}, nil
		}
		return entities.Estimate{}, err
	}
	if len(out.Attributes) == 0 {
		return entities.Estimate{}, nil
	}
	var it estimateItem
	if err := attributevalue.UnmarshalMap(out.Attributes, &it); err != nil {
		return entities.Estimate{}, err
	}
	return fromEstimateItem(it), nil
}

func toEstimateItem(e entities.Estimate) estimateItem {
	return estimateItem{
		ID:        e.ID,
		OSID:      e.OSID,
		Price:     floatToString(e.Price),
		Status:    string(e.Status),
		CreatedAt: e.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: e.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func fromEstimateItem(it estimateItem) entities.Estimate {
	createdAt, _ := time.Parse(time.RFC3339Nano, it.CreatedAt)
	updatedAt, _ := time.Parse(time.RFC3339Nano, it.UpdatedAt)
	price, _ := strconv.ParseFloat(it.Price, 64)
	return entities.Estimate{
		ID:        it.ID,
		OSID:      it.OSID,
		Price:     price,
		Status:    entities.EstimateStatus(it.Status),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func floatToString(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func mergeNames(a, b map[string]string) map[string]string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	out := make(map[string]string, len(a)+len(b))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		out[k] = v
	}
	return out
}
