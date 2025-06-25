package utils

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	//"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// uploadToS3 envia o arquivo para o bucket S3 da AWS e retorna a URL pública.
func UploadToS3(file multipart.File, filename, bucket string) (string, error) {
	// Carrega a configuração da AWS (região + credenciais)
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(os.Getenv("AWS_REGION")),
	)
	if err != nil {
		return "", fmt.Errorf("erro ao carregar config AWS: %w", err)
	}

	// Cria cliente e uploader S3
	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	// Faz o upload
	result, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String("doacoes/" + filename),
		Body:   file,
		//ACL:    types.ObjectCannedACLPublicRead, // Torna o arquivo público
	})
	if err != nil {
		return "", fmt.Errorf("erro ao subir para o S3: %w", err)
	}

	// Retorna a URL pública do arquivo
	return result.Location, nil
}
