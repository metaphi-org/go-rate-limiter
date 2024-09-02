package datastore

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoDBDatastore struct {
	svc       *dynamodb.Client
	tableName string
	pkAttr    string // Attribute name for the Primary Key
	ttlAttr   string // Attribute name for the TTL (time-to-live)
	countAttr string // Attribute name for the count
}

func NewDynamoDBDatastore(cfg aws.Config, tableName, pkAttr, ttlAttr, countAttr string) *dynamoDBDatastore {
	return &dynamoDBDatastore{
		svc:       dynamodb.NewFromConfig(cfg),
		tableName: tableName,
		pkAttr:    pkAttr,
		ttlAttr:   ttlAttr,
		countAttr: countAttr,
	}
}

func (d *dynamoDBDatastore) IncrKeys(ctx context.Context, keys []KeyConfig) ([]int, []error) {
	incrementCounts := make([]int, len(keys))
	errs := make([]error, len(keys))
	wg := sync.WaitGroup{}

	for i, keyConfig := range keys {
		wg.Add(1)
		go func(i int, keyConfig KeyConfig) {
			defer wg.Done()

			ttl := time.Now().Add(keyConfig.MaxLifespan).Unix()

			// Define the update expression
			update := expression.Add(
				expression.Name(d.countAttr),
				expression.Value(1),
			).Set(
				expression.Name(d.ttlAttr),
				expression.IfNotExists(expression.Name(d.ttlAttr), expression.Value(ttl)),
			)

			// Build the DynamoDB expression
			expr, err := expression.NewBuilder().
				WithUpdate(update).
				Build()
			if err != nil {
				errs[i] = fmt.Errorf("failed to build DynamoDB expression: %v", err)
				return
			}

			// Create the UpdateItem input
			input := &dynamodb.UpdateItemInput{
				TableName: aws.String(d.tableName),
				Key: map[string]types.AttributeValue{
					d.pkAttr: &types.AttributeValueMemberS{Value: keyConfig.Key},
				},
				UpdateExpression:          expr.Update(),
				ConditionExpression:       expr.Condition(),
				ExpressionAttributeNames:  expr.Names(),
				ExpressionAttributeValues: expr.Values(),
				ReturnValues:              types.ReturnValueUpdatedNew,
			}

			// Execute the UpdateItem request
			result, err := d.svc.UpdateItem(ctx, input)
			if err != nil {
				errs[i] = fmt.Errorf("failed to update item: %v", err)
				return
			}

			// Extract the new count from the result
			count := 0
			if err := attributevalue.Unmarshal(result.Attributes[d.countAttr], &count); err != nil {
				errs[i] = fmt.Errorf("failed to unmarshal count attribute: %v", err)
				return
			}

			incrementCounts[i] = count
		}(i, keyConfig)
	}

	wg.Wait()

	return incrementCounts, errs
}
