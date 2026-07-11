package shared

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
func (m *S3Manager) UploadFileToS3(ctx context.Context, key string, fileBody io.Reader) error {
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

func (m *S3Manager) UploadDirToS3(ctx context.Context, keyPrefix string, dirPath string) error {

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// only upload regular files
		if !d.Type().IsRegular() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			file.Close()
			return err
		}

		s3Key := filepath.ToSlash(filepath.Join(keyPrefix, relPath))

		err = m.UploadFileToS3(ctx, s3Key, file)
		closeErr := file.Close()

		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (m *S3Manager) DownloadFileFromS3(
	ctx context.Context, bucket string, key string, ofileName string,
) error {
	result, err := m.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	defer result.Body.Close()

	file, err := os.Create(ofileName)
	if err != nil {
		return fmt.Errorf("Couldn't create file %v. Reason: %v\n", ofileName, err)
	}
	defer file.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return fmt.Errorf("Couldn't read object body from %v. Reason: %v\n", key, err)
	}
	_, err = file.Write(body)
	return err
}

func (m *S3Manager) DownloadDirFromS3(
	ctx context.Context, bucket string, keyPrefix string, ofileDir string,
) error {
	paginator := s3.NewListObjectsV2Paginator(m.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(keyPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return err
		}

		for _, obj := range page.Contents {
			switch path.Base(*obj.Key) {
			case "in.txt":
				if err := m.DownloadFileFromS3(ctx, bucket, *obj.Key, ofileDir+"in.txt"); err != nil {
					return err
				}

			case "out.txt":
				if err := m.DownloadFileFromS3(ctx, bucket, *obj.Key, ofileDir+"out.txt"); err != nil {
					return err
				}

			}
		}
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
