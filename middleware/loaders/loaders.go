package dataloader

import (
	"context"
	"fmt"
	"time"

	"github.com/DGISsoft/DGISback/models"
	"github.com/DGISsoft/DGISback/services/mongo"
	"github.com/graph-gophers/dataloader"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Loaders struct {
	UserLoader         *dataloader.Loader
	MarkerLoader       *dataloader.Loader
	NotificationLoader *dataloader.Loader
	ReportLoader       *dataloader.Loader
	UserNotificationLoader *dataloader.Loader
}

func NewLoaders(
	userService *mongo.UserService,
	markerService *mongo.MarkerService,
	notificationService *mongo.NotificationService,
	reportService *mongo.ReportService,
) *Loaders {
	return &Loaders{
		UserLoader:         newUserLoader(userService),
		MarkerLoader:       newMarkerLoader(markerService),
		NotificationLoader: newNotificationLoader(notificationService),
		ReportLoader:       newReportLoader(reportService),
		UserNotificationLoader: newUserNotificationLoader(notificationService),
	}
}

// ========== USER LOADER ==========

func newUserLoader(service *mongo.UserService) *dataloader.Loader {
	return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Преобразуем ключи в ObjectID
		ids := make([]primitive.ObjectID, len(keys))
		for i, key := range keys {
			id, err := primitive.ObjectIDFromHex(key.String())
			if err != nil {
				return resultsWithError(len(keys), fmt.Errorf("invalid user ID: %s", key.String()))
			}
			ids[i] = id
		}

		// Создаем фильтр для поиска пользователей по массиву ID
		filter := primitive.M{"_id": primitive.M{"$in": ids}}
		
		// Получаем пользователей
		users, err := service.FindUsers(ctx, filter)
		if err != nil {
			return resultsWithError(len(keys), err)
		}

		// Создаем маппинг пользователей по ID
		userMap := make(map[primitive.ObjectID]*models.User)
		for _, user := range users {
			userMap[user.ID] = user
		}

		// Формируем результаты в том же порядке, что и ключи
		results := make([]*dataloader.Result, len(keys))
		for i, id := range ids {
			if user, exists := userMap[id]; exists {
				results[i] = &dataloader.Result{Data: user}
			} else {
				results[i] = &dataloader.Result{Error: fmt.Errorf("user not found: %s", id.Hex())}
			}
		}
		return results
	}, dataloader.WithWait(2*time.Millisecond))
}

// ========== MARKER LOADER ==========

func newMarkerLoader(service *mongo.MarkerService) *dataloader.Loader {
	return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Преобразуем ключи в ObjectID
		ids := make([]primitive.ObjectID, len(keys))
		for i, key := range keys {
			id, err := primitive.ObjectIDFromHex(key.String())
			if err != nil {
				return resultsWithError(len(keys), fmt.Errorf("invalid marker ID: %s", key.String()))
			}
			ids[i] = id
		}

		// Получаем маркеры
		markers, err := service.GetAllMarkersWithUsers(ctx) // Используем метод с пользователями
		if err != nil {
			return resultsWithError(len(keys), err)
		}

		// Создаем маппинг маркеров по ID
		markerMap := make(map[primitive.ObjectID]*models.Marker)
		for _, marker := range markers {
			markerMap[marker.ID] = marker
		}

		// Формируем результаты в том же порядке, что и ключи
		results := make([]*dataloader.Result, len(keys))
		for i, id := range ids {
			if marker, exists := markerMap[id]; exists {
				results[i] = &dataloader.Result{Data: marker}
			} else {
				results[i] = &dataloader.Result{Error: fmt.Errorf("marker not found: %s", id.Hex())}
			}
		}
		return results
	}, dataloader.WithWait(2*time.Millisecond))
}

// ========== NOTIFICATION LOADER ==========

func newNotificationLoader(service *mongo.NotificationService) *dataloader.Loader {
	return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Преобразуем ключи в ObjectID
		ids := make([]primitive.ObjectID, len(keys))
		for i, key := range keys {
			id, err := primitive.ObjectIDFromHex(key.String())
			if err != nil {
				return resultsWithError(len(keys), fmt.Errorf("invalid notification ID: %s", key.String()))
			}
			ids[i] = id
		}

		// Загружаем уведомления по одному (так как нет метода для множественной загрузки)
		results := make([]*dataloader.Result, len(keys))
		for i, id := range ids {
			notification, err := service.GetNotificationByID(ctx, id)
			if err != nil {
				results[i] = &dataloader.Result{Error: err}
			} else {
				results[i] = &dataloader.Result{Data: notification}
			}
		}
		return results
	}, dataloader.WithWait(2*time.Millisecond))
}

// ========== USER NOTIFICATION LOADER ==========

func newUserNotificationLoader(service *mongo.NotificationService) *dataloader.Loader {
	return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Преобразуем ключи в ObjectID
		ids := make([]primitive.ObjectID, len(keys))
		for i, key := range keys {
			id, err := primitive.ObjectIDFromHex(key.String())
			if err != nil {
				return resultsWithError(len(keys), fmt.Errorf("invalid user notification ID: %s", key.String()))
			}
			ids[i] = id
		}

		// Загружаем пользовательские уведомления по одному
		results := make([]*dataloader.Result, len(keys))
		for i, id := range ids {
			userNotification, err := service.GetUserNotificationByID(ctx, id)
			if err != nil {
				results[i] = &dataloader.Result{Error: err}
			} else {
				results[i] = &dataloader.Result{Data: userNotification}
			}
		}
		return results
	}, dataloader.WithWait(2*time.Millisecond))
}

// ========== REPORT LOADER (по userID) ==========

func newReportLoader(service *mongo.ReportService) *dataloader.Loader {
	return dataloader.NewBatchedLoader(func(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
		// Преобразуем ключи в ObjectID
		userIDs := make([]primitive.ObjectID, len(keys))
		for i, key := range keys {
			id, err := primitive.ObjectIDFromHex(key.String())
			if err != nil {
				return resultsWithError(len(keys), fmt.Errorf("invalid user ID: %s", key.String()))
			}
			userIDs[i] = id
		}

		// Загружаем отчеты по каждому пользователю
		results := make([]*dataloader.Result, len(keys))
		for i, userID := range userIDs {
			// Получаем отчеты пользователя (ограничим 100 последними)
			reports, err := service.GetWeeklyReportsByUserID(ctx, userID, 100, 0)
			if err != nil {
				results[i] = &dataloader.Result{Error: err}
			} else {
				results[i] = &dataloader.Result{Data: reports}
			}
		}
		return results
	}, dataloader.WithWait(2*time.Millisecond))
}

// ========== HELPERS ==========

func resultsWithError(count int, err error) []*dataloader.Result {
	results := make([]*dataloader.Result, count)
	for i := range results {
		results[i] = &dataloader.Result{Error: err}
	}
	return results
}

// ========== CONTEXT HELPERS ==========

// Для получения Loaders из контекста
type ctxKey string

const loadersKey ctxKey = "dataloaders"

func For(ctx context.Context) *Loaders {
	return ctx.Value(loadersKey).(*Loaders)
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey, loaders)
}

// ========== CUSTOM KEY HELPERS ==========

// StringKey создает ключ даталоадера из строки
func StringKey(s string) dataloader.Key {
	return dataloader.StringKey(s)
}

// ObjectIDKey создает ключ даталоадера из ObjectID
func ObjectIDKey(id primitive.ObjectID) dataloader.Key {
	return dataloader.StringKey(id.Hex())
}