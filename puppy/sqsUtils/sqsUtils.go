package sqsUtils

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type EnvVars struct {
	QueueUrl string
	Region   string
	Ec2_api  string
}

func ConnectToSqs(region string) (*sqs.Client, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(
			credentials.StaticCredentialsProvider{
				Value: aws.Credentials{
					AccessKeyID:     "",
					SecretAccessKey: "",
					SessionToken:    "",
					Source:          "linux/local",
				},
			},
		),
		config.WithRegion(region),
	)

	//cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	svc := sqs.NewFromConfig(cfg)
	return svc, nil

}

func SendToSqs(QueueUrl string, svc *sqs.Client, MessageGroupId string) (*sqs.SendMessageOutput, error) {

	// var MessageGroupId = "MGS-d"
	//Send message
	send_params := &sqs.SendMessageInput{
		MessageBody:    aws.String("2"),
		QueueUrl:       aws.String(QueueUrl),
		MessageGroupId: &MessageGroupId,
	}
	send_resp, err := svc.SendMessage(context.TODO(), send_params)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("[Send message] \n%v \n\n", aws.ToString(send_resp.MessageId))
	return send_resp, nil

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

	file, err := os.Open(".cutsom.env")
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
	return EnvVars{
		QueueUrl: env_map["SQS_QueueUrl"],
		Region:   env_map["SQS_REGION"],
		Ec2_api:  env_map["EC2_API"],
	}
}
