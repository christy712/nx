package s3utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// s3ClientSingleton holds the singleton S3 client and related metadata
type s3ClientSingleton struct {
	client      *s3.Client
	lastChecked time.Time
	isHealthy   bool
	mutex       sync.RWMutex
}

var (
	// singleton instance
	singleton *s3ClientSingleton
	// ensure singleton is initialized only once
	once sync.Once
)

// getS3Client returns the singleton S3 client, initializing it if necessary
func getS3Client(region string) (*s3.Client, error) {
	once.Do(func() {
		singleton = &s3ClientSingleton{
			isHealthy:   false,
			lastChecked: time.Time{},
		}
	})

	singleton.mutex.RLock()
	// Check if client exists and is healthy, or if health check is overdue
	if singleton.client != nil && singleton.isHealthy && time.Since(singleton.lastChecked) < 5*time.Minute {
		defer singleton.mutex.RUnlock()
		return singleton.client, nil
	}
	singleton.mutex.RUnlock()

	// Acquire write lock to regenerate client
	singleton.mutex.Lock()
	defer singleton.mutex.Unlock()

	// Double-check to avoid race conditions
	if singleton.client != nil && singleton.isHealthy && time.Since(singleton.lastChecked) < 5*time.Minute {
		return singleton.client, nil
	}

	// Initialize new S3 client

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

	// 	config.WithRegion(region))
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))

	if err != nil {
		singleton.isHealthy = false
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := s3.NewFromConfig(cfg)
	// Verify client health
	if err := checkClientHealth(client); err != nil {
		singleton.isHealthy = false
		return nil, fmt.Errorf("client health check failed: %v", err)
	}

	singleton.client = client
	singleton.isHealthy = true
	singleton.lastChecked = time.Now()

	return singleton.client, nil
}

// checkClientHealth verifies if the S3 client is active by listing buckets
func checkClientHealth(client *s3.Client) error {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	return nil
}

// PutObject uploads a file to S3 and returns the public URL
func PutObject(bucket string, key string, file io.Reader, region string) (string, error) {
	client, err := getS3Client(region)
	if err != nil {
		return "", err
	}

	// file, err := os.Open(filePath)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to open file: %v", err)
	// }
	//defer file.Close()

	_, err = client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		singleton.mutex.Lock()
		singleton.isHealthy = false
		singleton.mutex.Unlock()
		return "", fmt.Errorf("failed to upload object: %v", err)
	}

	publicURL := fmt.Sprintf(
		"https://%s.s3.%s.amazonaws.com/%s",
		bucket, region, key,
	)
	return publicURL, nil
}

// GetObject downloads a file from S3 to the specified file path
func GetObject(bucket, key, filePath, region string) error {
	client, err := getS3Client(region)
	if err != nil {
		return err
	}

	downloader := manager.NewDownloader(client)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = downloader.Download(context.TODO(), file, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		singleton.mutex.Lock()
		singleton.isHealthy = false
		singleton.mutex.Unlock()
		return fmt.Errorf("failed to download object: %v", err)
	}

	return nil
}
