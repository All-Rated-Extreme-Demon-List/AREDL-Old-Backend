package util

func UseOtherIfNil[T comparable](value interface{}, other T) interface{} {
	if value == nil {
		return other
	}
	return value
}
