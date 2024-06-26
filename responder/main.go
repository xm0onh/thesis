package main

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	blockchainPkg "github.com/xm0onh/thesis/packages/blockchain"
	utils "github.com/xm0onh/thesis/packages/utils"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go/aws"
)

var ddbTableName = os.Getenv("DDB_TABLE_NAME")
var setupTableName = os.Getenv("SETUP_DB")
var responderID = os.Getenv("RESPONDER_ID")
var bucketName = os.Getenv("BLOCKCHAIN_S3_BUCKET")

func init() {
	gob.Register(blockchainPkg.Transaction{})
	gob.Register(blockchainPkg.Block{})
}

func Handler(ctx context.Context, snsEvent events.SNSEvent) error {
	fmt.Println("I'm responder: ", responderID)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to load AWS configuration, %w", err)
	}
	param := utils.SetupParameters{}
	param.DegreeCDF, param.SourceBlocks, param.EncodedBlockIDs, param.RandomSeed, param.NumberOfBlocks, _, param.MessageSize, err = utils.PullDataFromSetup(ctx, setupTableName)
	if err != nil {
		fmt.Printf("Failed to pull data from setup: %v\n", err)
		return err
	}
	startTime := time.Now()
	param.Message, _ = utils.DownloadFromS3(ctx, bucketName, "blockchain_data")
	fmt.Println("Time to download blockchain data: ", time.Since(startTime))

	ddbClient := dynamodb.NewFromConfig(cfg)
	for _, record := range snsEvent.Records {
		var dropletReq utils.RequestedDroplets

		err := json.Unmarshal([]byte(record.SNS.Message), &dropletReq)
		fmt.Println("Received request for block numbers: ", dropletReq)
		if err != nil {
			fmt.Printf("Failed to unmarshal LTBlock data: %v\n", err)
			continue
		}
		droplets := utils.GenerateDroplet(param)
		fmt.Println("Generated droplets: ", len(droplets))

		// Uploading only the droplets within the range of start and end
		for i := dropletReq.Start; i < dropletReq.End; i++ {
			_, err = ddbClient.PutItem(ctx, &dynamodb.PutItemInput{
				TableName: aws.String(ddbTableName),
				Item: map[string]types.AttributeValue{
					"ID":        &types.AttributeValueMemberS{Value: strconv.Itoa(i)},
					"Data":      &types.AttributeValueMemberB{Value: droplets[i].Data},
					"BlockCode": &types.AttributeValueMemberN{Value: strconv.FormatInt(droplets[i].BlockCode, 10)},
				},
				ConditionExpression: aws.String("attribute_not_exists(ID)"),
			})

			if err != nil {
				fmt.Printf("Skip the droplet because it's already exists: %v\n", err)
				continue
			}
		}

	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
