package dispatcher

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type SubmissionSpec struct {
	SubmissionID   string `json:"submission_id"`
	Language       string `json:"language"`
	Version        string `json:"version"`
	Source         string `json:"source"`
	Testset        string `json:"testset"`
	TestsetVersion string `json:"testset_version"`
}

type JobSpec struct {
	JobId          string `json:"job_id"`
	SubmissionID   string `json:"submission_id"`
	Language       string `json:"language"`
	Version        string `json:"version"`
	S3Key          string `json:"s3key"`
	Testset        string `json:"testset"`
	TestsetVersion string `json:"testset_version"`
}

type S3Manager struct {
	client *s3.Client
	bucket string
}

func InitS3Manager(
	ctx context.Context, bucket, region, accessKey, secretKey, customEndpoint string,
) (*S3Manager, error) {
	credProvider := credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
		config.WithCredentialsProvider(credProvider),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to load SDK config: %w", err)
	}

	// create S3 client
	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		// if a custom endpoint is passed, reroute traffic locally
		if customEndpoint != "" {
			o.BaseEndpoint = aws.String(customEndpoint)
			o.UsePathStyle = true // MinIO requires path-style addressing (bucket/key)
		}
	})

	return &S3Manager{
		client: s3Client,
		bucket: bucket,
	}, nil
}

// UploadToS3 streams an item up into the configured storage instance
func (m *S3Manager) UploadToS3(ctx context.Context, key string, fileBody io.Reader) error {
	_, err := m.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(m.bucket),
		Key:    aws.String(key),
		Body:   fileBody,
	})

	if err != nil {
		return fmt.Errorf("failed to upload object to storage: %w", err)
	}

	return nil
}

// CheckS3Dir scans for the existence of a virtual directory prefix
func (m *S3Manager) CheckS3Dir(ctx context.Context, dirPath string) (bool, error) {

	if !strings.HasSuffix(dirPath, "/") {
		dirPath = dirPath + "/"
	}

	output, err := m.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket:  aws.String(m.bucket),
		Prefix:  aws.String(dirPath),
		MaxKeys: aws.Int32(1), // exit instantly if a single file exists
	})

	if err != nil {
		return false, fmt.Errorf("failed to list keys for directory check: %w", err)
	}

	// if KeyCount is greater than 0, the prefix contains objects
	return *output.KeyCount > 0, nil
}

func IsLanguageSupported(lang, version string) bool {
	versions, ok := AvailableLanguages[lang]
	if !ok {
		return false
	}

	_, ok = versions[version]
	return ok
}
