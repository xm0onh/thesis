package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	lubyTransform "github.com/xm0onh/thesis/packages/luby"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

// var snsTopicARN = os.Getenv("STARTER_SNS_TOPIC_ARN")
var tableName = os.Getenv("SETUP_DB")

// var snsClient *sns.Client

type StartSignal struct {
	Start           bool `json:"start"`
	SourceBlocks    int  `json:"sourceBlocks"`
	EncodedBlockIDs int  `json:"encodedBlockIDs"`
}

func Handler(ctx context.Context, event StartSignal) (string, error) {
	if !event.Start {
		return "Event does not contain start signal", nil
	}
	sourceBlocks := event.SourceBlocks
	encodedBlockIDs := event.EncodedBlockIDs
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to load AWS configuration, %w", err)
	}

	ddbClient := dynamodb.NewFromConfig(cfg)

	// Type of degreeCDF is []float64
	degreeCDF := lubyTransform.SolitonDistribution(sourceBlocks)

	degreeCDFString, _ := json.Marshal(degreeCDF)
	// Create a PRNG source.
	seed := time.Now().UnixNano()

	// Add MessageSize into the Database
	_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"ID":              &types.AttributeValueMemberS{Value: "message"},
			"degreeCDF":       &types.AttributeValueMemberS{Value: string(degreeCDFString)},
			"randomSeed":      &types.AttributeValueMemberN{Value: strconv.FormatInt(seed, 10)},
			"sourceBlocks":    &types.AttributeValueMemberN{Value: strconv.Itoa(sourceBlocks)},
			"encodedBlockIDs": &types.AttributeValueMemberN{Value: strconv.Itoa(encodedBlockIDs)},
		},
	})
	if err != nil {

		return "Failed to put metadata item into DynamoDB", err
	}
	return "Hello, World!", nil
}

func main() {
	lambda.Start(Handler)
}
