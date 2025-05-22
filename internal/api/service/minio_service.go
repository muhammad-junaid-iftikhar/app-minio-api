package service

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
)

type MinioService struct {
	MinioClient *minio.Client
	BucketName  string
}

func NewMinioService(client *minio.Client, bucketName string) *MinioService {
	return &MinioService{
		MinioClient: client,
		BucketName:  bucketName,
	}
}

func (s *MinioService) UploadFile(ctx context.Context, objectName string, file io.Reader, size int64, contentType string) (*minio.UploadInfo, error) {
	return s.MinioClient.PutObject(ctx, s.BucketName, objectName, file, size, minio.PutObjectOptions{ContentType: contentType})
}

func (s *MinioService) ListFiles(ctx context.Context) ([]minio.ObjectInfo, error) {
	objectCh := s.MinioClient.ListObjects(ctx, s.BucketName, minio.ListObjectsOptions{Recursive: true})
	var objects []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return nil, object.Err
		}
		objects = append(objects, object)
	}
	return objects, nil
}

func (s *MinioService) GetFile(ctx context.Context, objectName string) (*minio.Object, error) {
	return s.MinioClient.GetObject(ctx, s.BucketName, objectName, minio.GetObjectOptions{})
}

func (s *MinioService) StatFile(ctx context.Context, object *minio.Object) (minio.ObjectInfo, error) {
	return object.Stat()
}

func (s *MinioService) DeleteFile(ctx context.Context, objectName string) error {
	return s.MinioClient.RemoveObject(ctx, s.BucketName, objectName, minio.RemoveObjectOptions{})
}

func (s *MinioService) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return s.MinioClient.ListBuckets(ctx)
}
