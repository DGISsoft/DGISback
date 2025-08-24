package mongo

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/DGISsoft/DGISback/env"
	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo/query"
	"github.com/DGISsoft/DGISback/services/s3"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ReportService struct {
	*MongoService
}

func NewReportService(mongoService *MongoService) *ReportService {
	return &ReportService{MongoService: mongoService}
}

func (s *ReportService) GetCollection() *mongo.Collection {
	return s.MongoService.GetCollection("weekly_reports")
}

func (s *ReportService) CreateWeeklyReport(
	ctx context.Context,
	report *models.WeeklyReport,
) (*models.WeeklyReport, error) {
	collection := s.GetCollection()

	now := time.Now()
	report.CreatedAt = now
	report.UpdatedAt = now

	if report.Status == "" {
		report.Status = models.StatusNotReviewed
	}

	res, err := collection.InsertOne(ctx, report)
	if err != nil {
		return nil, fmt.Errorf("failed to create weekly report: %w", err)
	}

	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		report.ID = oid
		log.Printf("ReportService: Created report with ID %s", report.ID.Hex())
	} else {
		return nil, fmt.Errorf("failed to get inserted report ID, expected ObjectID, got %T", res.InsertedID)
	}

	return report, nil
}

func (s *ReportService) GetWeeklyReportByID(
	ctx context.Context,
	id primitive.ObjectID,
) (*models.WeeklyReport, error) {
	collection := s.GetCollection()

	var report models.WeeklyReport

	err := query.FindByID(ctx, collection, id, &report)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("weekly report not found")
		}
		return nil, fmt.Errorf("failed to get weekly report: %w", err)
	}

	return &report, nil
}

func (s *ReportService) GetWeeklyReports(
	ctx context.Context,
	filter bson.M,
	limit, skip int64,
) ([]*models.WeeklyReport, error) {
	collection := s.GetCollection()

	var reports []*models.WeeklyReport

	err := query.FindWithPagination(ctx, collection, filter, &reports, limit, skip)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly reports: %w", err)
	}

	return reports, nil
}

func (s *ReportService) GetWeeklyReportsByUserID(
	ctx context.Context,
	userID primitive.ObjectID,
	limit, skip int64,
) ([]*models.WeeklyReport, error) {
	filter := bson.M{"user_id": userID}
	return s.GetWeeklyReports(ctx, filter, limit, skip)
}

func (s *ReportService) UpdateWeeklyReport(
	ctx context.Context,
	id primitive.ObjectID,
	updateData bson.M,
) error {
	collection := s.GetCollection()

	updateQuery := bson.M{
		"$set":         updateData,
		"$currentDate": bson.M{"updated_at": true},
	}

	_, err := collection.UpdateOne(ctx, bson.M{"_id": id}, updateQuery)
	if err != nil {
		return fmt.Errorf("failed to update weekly report: %w", err)
	}

	return nil
}

func (s *ReportService) UpdateReportStatus(
	ctx context.Context,
	id primitive.ObjectID,
	status models.ReportStatus,
) error {
	return s.UpdateWeeklyReport(ctx, id, bson.M{"status": status})
}

func (s *ReportService) SetSupervisorRate(
	ctx context.Context,
	id primitive.ObjectID,
	rate models.Rating,
) error {
	if rate != models.RatingGood && rate != models.RatingBad {
		return fmt.Errorf("invalid rating value, must be 'good' or 'bad'")
	}
	return s.UpdateWeeklyReport(ctx, id, bson.M{"supervisor_rate": rate})
}

func (s *ReportService) SetPredsedatelRate(
	ctx context.Context,
	id primitive.ObjectID,
	rate models.Rating,
) error {
	if rate != models.RatingGood && rate != models.RatingBad {
		return fmt.Errorf("invalid rating value, must be 'good' or 'bad'")
	}
	return s.UpdateWeeklyReport(ctx, id, bson.M{"predsedatel_rate": rate})
}

func (s *ReportService) DeleteWeeklyReport(
	ctx context.Context,
	bucket string, // Добавлен параметр bucket
	id primitive.ObjectID,
) error {
	collection := s.GetCollection()

	report, err := s.GetWeeklyReportByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get report for deletion: %w", err)
	}

	// Используем переданный bucket
	if s3.Service != nil {
		if bucket == "" {
			bucket = env.GetEnv("S3_BUCKET", "your-default-bucket")
		}
		for _, key := range report.ApplicationsImageKeys {
			s3.Service.DeleteFile(ctx, bucket, key)
		}
		for _, key := range report.InspectionImageKeys {
			s3.Service.DeleteFile(ctx, bucket, key)
		}
		for _, key := range report.AdditionalImageKeys {
			s3.Service.DeleteFile(ctx, bucket, key)
		}
	}

	_, err = collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete weekly report: %w", err)
	}

	log.Printf("ReportService: Deleted report with ID %s", id.Hex())
	return nil
}

func (s *ReportService) AddImageKeysToReport(
	ctx context.Context,
	id primitive.ObjectID,
	appKeys, inspKeys, addKeys []string,
) error {
	updateData := bson.M{}
	
	if len(appKeys) > 0 {
		updateData["applications_image_keys"] = bson.M{"$each": appKeys}
	}
	if len(inspKeys) > 0 {
		updateData["inspection_image_keys"] = bson.M{"$each": inspKeys}
	}
	if len(addKeys) > 0 {
		updateData["additional_image_keys"] = bson.M{"$each": addKeys}
	}
	
	if len(updateData) == 0 {
		return nil
	}
	
	collection := s.GetCollection()
	_, err := collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$push": updateData},
	)
	
	if err != nil {
		return fmt.Errorf("failed to add image keys to report: %w", err)
	}
	
	return nil
}

func (s *ReportService) UploadImage(
	ctx context.Context,
	bucket string, // Добавлен параметр bucket
	fileContent []byte,
	fileName string,
	contentType string,
) (string, error) {
	if s3.Service == nil {
		return "", fmt.Errorf("S3 service not initialized")
	}

	// Используем переданный bucket
	if bucket == "" {
		bucket = env.GetEnv("S3_BUCKET", "your-default-bucket")
	}

	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().Unix(), fileName)

	err := s3.Service.UploadFile(ctx, bucket, uniqueFileName, fileContent, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload image to S3: %w", err)
	}

	log.Printf("ReportService: Uploaded image %s to S3", uniqueFileName)
	return uniqueFileName, nil
}

func (s *ReportService) GetImage(
	ctx context.Context,
	bucket string, // Добавлен параметр bucket
	key string,
) ([]byte, error) {
	if s3.Service == nil {
		return nil, fmt.Errorf("S3 service not initialized")
	}

	// Используем переданный bucket
	if bucket == "" {
		bucket = env.GetEnv("S3_BUCKET", "your-default-bucket")
	}

	content, err := s3.Service.DownloadFile(ctx, bucket, key)
	if err != nil {
		return nil, fmt.Errorf("failed to download image from S3: %w", err)
	}

	return content, nil
}