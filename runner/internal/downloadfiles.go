package internal

import (
	"context"
	"shared"
)

// convenient wrapper over S3Manager's methods
func downloadFileS3(
	ctx context.Context, s3m shared.S3Manager,
	bucket string, testsetS3key string,
	hostSrcFilePath string, hostTestFileDir string,
) error {

	if err := s3m.DownloadDirFromS3(ctx, bucket, testsetS3key, hostTestFileDir); err != nil {
		return err
	}

	return nil
}
