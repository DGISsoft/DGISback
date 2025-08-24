package s3

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

type S3ClientConfig struct {
	Bucket    string
	Endpoint  string
	Region    string
	AccessKey string
	SecretKey string
}

type S3ConfigProvider interface {
	GetBucket() string
	GetEndpoint() string
	GetRegion() string
	GetAccessKey() string
	GetSecretKey() string
}

func (cfg *S3ClientConfig) GetBucket() string {
	return cfg.Bucket
}

func (cfg *S3ClientConfig) GetEndpoint() string {
	return cfg.Endpoint
}

func (cfg *S3ClientConfig) GetRegion() string {
	return cfg.Region
}

func (cfg *S3ClientConfig) GetAccessKey() string {
	return cfg.AccessKey
}

func (cfg *S3ClientConfig) GetSecretKey() string {
	return cfg.SecretKey
}

func NewS3Config(s3cfg S3ConfigProvider) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(s3cfg.GetRegion()), config.WithBaseEndpoint(s3cfg.GetEndpoint()),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3cfg.GetAccessKey(),
			s3cfg.GetSecretKey(),
			"",
		)))
	return cfg, err
}
