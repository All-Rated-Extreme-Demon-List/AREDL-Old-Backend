package util

import (
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/daos"
	"github.com/pocketbase/pocketbase/models"
	"math/rand"
)

func UseOtherIfNil[T comparable](value interface{}, other T) interface{} {
	if value == nil {
		return other
	}
	return value
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func CreatePlaceholderUser(app *pocketbase.PocketBase, dao *daos.Dao, userCollection *models.Collection, name string) (*models.Record, error) {
	password := RandString(20)
	usedName := RandString(10)
	userRecord, err := AddRecord(dao, app, userCollection, map[string]any{
		"username":    usedName,
		"permissions": "member",
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
