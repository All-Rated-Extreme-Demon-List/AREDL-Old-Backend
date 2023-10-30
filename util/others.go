package util

import (
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"math/rand"
	"net/http"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func CreatePlaceholderUser(app core.App, dao *daos.Dao, userCollection *models.Collection, name string) (*models.Record, error) {
	password := RandString(20)
	usedName := RandString(10)
	userRecord, err := AddRecord(dao, app, userCollection, map[string]any{
		"username":    usedName,
		"role":        "member",
		"global_name": name,
		"placeholder": true,
		//"email":           usedName + "@none.com",
		"password":        password,
		"passwordConfirm": password,
	})
	return userRecord, err
}

// MapSlice maps a slice using the mapper function
func MapSlice[T, U any](slice []T, mapper func(T) U) []U {
	result := make([]U, len(slice))
	for i := range slice {
		result[i] = mapper(slice[i])
	}
	return result
}

type ErrorResponse struct {
	apis.ApiError
}

func NewErrorResponse(err error, message string) error {
	if err == nil {
		return apis.NewApiError(http.StatusBadRequest, message, nil)
	}
	switch err.(type) {
	case validation.Errors:
		return apis.NewApiError(http.StatusBadRequest, "Invalid data: "+err.Error(), nil)
	default:
		return apis.NewApiError(http.StatusBadRequest, message+": "+err.Error(), nil)
	}
}
