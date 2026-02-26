package awsclient

import (
	"os"
	"context"
	"fmt"
	"mime/multipart"

	"github.com/joho/godotenv"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Service struct {
	Client *s3.Client
	Bucket string
	MediaBaseURL string
}

func (service *S3Service) UploadFile(ctx context.Context, DestinationPath string, file multipart.File, ContentType string) (string, error) {
	fmt.Printf("AWS: %s | %s", service.Bucket, service.MediaBaseURL)
	_, err := service.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(service.Bucket),
		Key: aws.String(DestinationPath),
		Body: file,
		ContentType: aws.String(ContentType),
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", service.MediaBaseURL, DestinationPath), nil
}

var Service *S3Service

func NewS3Service(region, bucket, mediaBaseURL string) *S3Service {
	cfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return nil
	}

	return &S3Service{
		Client: s3.NewFromConfig(cfg),
		Bucket: bucket,
		MediaBaseURL: mediaBaseURL,
	}
}

func Init() {
	_ = godotenv.Load()
	
	AWSRegion := os.Getenv("AWS_REGION")
	AWSS3Bucket := os.Getenv("AWS_S3_BUCKET")
	MediaBaseURL := os.Getenv("MEDIA_BASE_URL")
	
	Service = NewS3Service(AWSRegion, AWSS3Bucket, MediaBaseURL)
}
