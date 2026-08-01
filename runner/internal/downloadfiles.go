package internal

import (
	"context"
	"shared"
)

func downloadFileS3(
	ctx context.Context, s3m shared.S3Manager,
	bucket string, srcCodeS3key string, testsetS3key string,
	hostSrcFilePath string, hostTestFileDir string,
) error {

	if err := s3m.DownloadFileFromS3(ctx, bucket, srcCodeS3key, hostSrcFilePath); err != nil {
		return err
	}

	if err := s3m.DownloadDirFromS3(ctx, bucket, testsetS3key, hostTestFileDir); err != nil {
		return err
	}

	return nil
}
