package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Service struct {
	client *s3.Client
}


var Service *S3Service

func Init(bucket,endpoint,region,accceskey,secretkey string) {
	cfg := &S3ClientConfig{
		Bucket:    bucket,
		Endpoint:  endpoint,
		Region:    region,
		AccessKey: accceskey,
		SecretKey: secretkey,
	}
    svc, err := NewS3Service(cfg)
    if err != nil {
        fmt.Println("Ошибка подключения s3:", err)
        return
    }
    Service = svc   
    log.Printf("s3 подключен")
}

func NewS3Service(cfg S3ConfigProvider) (*S3Service, error) {

	configAWS, err := NewS3Config(cfg)
	if err != nil {
		return nil, fmt.Errorf("Ошибка загрузки конфига aws: %w", err)
	}

	client := s3.NewFromConfig(configAWS)
	return &S3Service{
		client: client,
	}, nil
}

func (s *S3Service) FileExists(ctx context.Context, bucket, fileName string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileName),
	})
	if err != nil {
		var awsErr *types.NotFound
		if errors.As(err, &awsErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *S3Service) DeleteFile(ctx context.Context, bucket, fileName string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileName),
	})
	return err
}

func (s *S3Service) UploadFile(ctx context.Context, bucket, fileName string, content []byte, contentType string) error {
	exists, err := s.FileExists(ctx, bucket, fileName)
	if err != nil {
		return err
	}

	if exists {
		if err := s.DeleteFile(ctx, bucket, fileName); err != nil {
			return fmt.Errorf("Ошибка при удалении файла %w", err)
		}
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(fileName),
		Body:        bytes.NewReader(content),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("Ошибка при загрузке файла %w", err)
	}

	log.Printf("Файл %s успешно загружен в бакет %s", fileName, bucket)
	return nil
}

func (s *S3Service) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
    out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    })
    if err != nil {
        return nil, fmt.Errorf("s3.GetObject failed: %w", err)
    }
    defer out.Body.Close()

    buf := new(bytes.Buffer)
    if _, err := io.Copy(buf, out.Body); err != nil {
        return nil, fmt.Errorf("read S3 object body: %w", err)
    }
    return buf.Bytes(), nil
}