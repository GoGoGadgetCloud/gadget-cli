package util

import (
	"context"
	"io"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

func MultipartUpload(ctx context.Context, client *s3.Client, bucket, key string, body io.Reader, partSize int64) error {
	// Create multipart upload
	createResp, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	partNum := int32(1)
	completedParts := []types.CompletedPart{}

	for {
		part := make([]byte, partSize)
		read, err := body.Read(part)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		wg.Add(1)
		go func(partNum int32, part []byte) {
			defer wg.Done()

			// Upload part
			uploadResp, err := client.UploadPart(ctx, &s3.UploadPartInput{
				Bucket:     aws.String(bucket),
				Key:        aws.String(key),
				PartNumber: &partNum,
				//Body:       aws.ReadSeekCloser(strings.NewReader(string(part))),
				UploadId: createResp.UploadId,
			})
			if err != nil {
				// Handle error
				return
			}

			completedParts = append(completedParts, types.CompletedPart{
				ETag:       uploadResp.ETag,
				PartNumber: &partNum,
			})
		}(partNum, part[:read])

		partNum++
	}

	wg.Wait()

	// Complete multipart upload
	_, err = client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(bucket),
		Key:      aws.String(key),
		UploadId: createResp.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
