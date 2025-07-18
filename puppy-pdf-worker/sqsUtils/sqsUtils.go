package sqsUtils

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type EnvVars struct {
	QueueUrl   string
	S3Region   string
	SqsRegion  string
	Ec2_api    string
	Bucket     string
	MaxThreads int
}

type sqsClientSingleton struct {
	client      *sqs.Client
	isHealthy   bool
	lastChecked time.Time
	mutex       sync.RWMutex
}

var (
	sqsOnce     sync.Once
	sqsInstance *sqsClientSingleton
)

func ConnectToSqs(region string) (*sqs.Client, error) {
	// cfg, err := config.LoadDefaultConfig(context.TODO(),
	// 	config.WithCredentialsProvider(
	// 		credentials.StaticCredentialsProvider{
	// 			Value: aws.Credentials{
	// 				AccessKeyID:     "",
	// 				SecretAccessKey: "",
	// 				SessionToken:    "",
	// 				Source:          "linux/local",
	// 			},
	// 		},
	// 	),
	// 	config.WithRegion(region),
	// )

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	svc := sqs.NewFromConfig(cfg)
	return svc, nil

}

func getSqsClient(region string) (*sqs.Client, error) {
	sqsOnce.Do(func() {
		sqsInstance = &sqsClientSingleton{
			isHealthy:   false,
			lastChecked: time.Time{},
		}
	})

	sqsInstance.mutex.RLock()
	if sqsInstance.client != nil && sqsInstance.isHealthy && time.Since(sqsInstance.lastChecked) < 5*time.Minute {
		defer sqsInstance.mutex.RUnlock()
		return sqsInstance.client, nil
	}
	sqsInstance.mutex.RUnlock()

	sqsInstance.mutex.Lock()
	defer sqsInstance.mutex.Unlock()

	if sqsInstance.client != nil && sqsInstance.isHealthy && time.Since(sqsInstance.lastChecked) < 5*time.Minute {
		return sqsInstance.client, nil
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		sqsInstance.isHealthy = false
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := sqs.NewFromConfig(cfg)

	// Optional: light health check
	if err := checkSqsHealth(client); err != nil {
		sqsInstance.isHealthy = false
		return nil, fmt.Errorf("SQS client health check failed: %v", err)
	}

	sqsInstance.client = client
	sqsInstance.isHealthy = true
	sqsInstance.lastChecked = time.Now()

	return sqsInstance.client, nil
}
func checkSqsHealth(client *sqs.Client) error {
	_, err := client.ListQueues(context.TODO(), &sqs.ListQueuesInput{
		MaxResults: aws.Int32(1),
	})
	return err
}

func SendToSqs(QueueUrl string, svc *sqs.Client, MessageGroupId string, Body string) (*sqs.SendMessageOutput, string, error) {

	// var MessageGroupId = "MGS-d"
	//Send message
	send_params := &sqs.SendMessageInput{
		MessageBody:    aws.String(Body),
		QueueUrl:       aws.String(QueueUrl),
		MessageGroupId: &MessageGroupId,
	}
	send_resp, err := svc.SendMessage(context.TODO(), send_params)
	if err != nil {
		return nil, "", err
	}
	// fmt.Printf("[Send message] \n%v \n\n", aws.ToString(send_resp.MessageId))
	return send_resp, aws.ToString(send_resp.MessageId), nil

}

// Poll mesages from queue , one at a time
func PollMessagesFromSqs(QueueUrl string, svc *sqs.Client) (*sqs.ReceiveMessageOutput, error) {

	receive_params := &sqs.ReceiveMessageInput{
		QueueUrl: aws.String(QueueUrl),
		// MaxNumberOfMessages: 3,
		// VisibilityTimeout:   30,
		// WaitTimeSeconds:     20,
	}
	receive_resp, err := svc.ReceiveMessage(context.TODO(), receive_params)
	if err != nil {
		return nil, err
	}
	return receive_resp, nil

}

// Delete Message from queue // This shouldnot be defer'd as this would mess with the DLQ logic
// Making ths in-accesible for main for now
func deleteMessageFromSqs(QueueUrl string, svc *sqs.Client, receive_resp *sqs.ReceiveMessageOutput) error {

	for _, message := range receive_resp.Messages {
		delete_params := &sqs.DeleteMessageInput{
			QueueUrl:      aws.String(QueueUrl),
			ReceiptHandle: message.ReceiptHandle,
		}
		_, err := svc.DeleteMessage(context.TODO(), delete_params)
		if err != nil {
			return err
		}
	}

	return nil

}

func LoadEnv() EnvVars {

	file, err := os.Open(".custom.env")
	if err != nil {

		log.Fatal("No .custom.env Found ")
		os.Exit(200)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	env_map := map[string]string{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines or comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split line into key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue // Skip malformed lines
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		env_map[key] = value

	}
	//fmt.Println(env_map)
	mt, err := strconv.Atoi(env_map["CPU_MAX_THREADS"])
	if err != nil || mt <= 0 {
		mt = runtime.NumCPU()
	}
	return EnvVars{
		QueueUrl:   env_map["SQS_QueueUrl"],
		SqsRegion:  env_map["SQS_REGION"],
		Ec2_api:    env_map["EC2_API"],
		Bucket:     env_map["S3_BUCKET"],
		S3Region:   env_map["S3_REGION"],
		MaxThreads: mt,
	}
}
