package micropub

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func List() ([]string, error) {
	bucketName := "micropub.funabashi.co.uk"
	postList := []string{}

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return postList, fmt.Errorf("failed to load aws config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	output, err := client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return postList, fmt.Errorf("failed to list objects: %v", err)
	}

	for _, object := range output.Contents {
		postList = append(postList, *object.Key)
	}

	return postList, nil
}

func Download(fileKey string) (io.ReadCloser, error) {
	bucketName := "micropub.funabashi.co.uk"
	ctx := context.Background()
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load aws config: %v", err)
	}

	client := s3.NewFromConfig(cfg)

	output, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &fileKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %v", err)
	}

	return output.Body, nil
}
