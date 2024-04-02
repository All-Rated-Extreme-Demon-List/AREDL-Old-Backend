package middlewares

import (
	"AREDL/util"
	"encoding/json"
	"fmt"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v5"
	"strconv"
)

type LoadFunc func(string, map[string][]string) (interface{}, error)
type LoadData map[string]LoadFunc

func LoadParam(toLoad LoadData) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			params, err := c.FormValues()
			if err != nil {
				return util.NewErrorResponse(err, "Failed to load form data")
			}
			for _, param := range c.PathParams() {
				params[param.Name] = []string{param.Value}
			}
			for key, loadFunc := range toLoad {
				value, err := loadFunc(key, params)
				if err != nil {
					return err
				}
				if value != nil {
					c.Set(key, value)
				}
			}
			return next(c)
		}
	}
}

func LoadString(required bool, rules ...validation.Rule) LoadFunc {
	// no conversion
	return loadWithConversion(func(v string) (string, error) { return v, nil }, "", required, rules...)
}

func LoadInt(required bool, rules ...validation.Rule) LoadFunc {
	return loadWithConversion(strconv.Atoi, " is not an int", required, rules...)
}

func LoadBool(required bool, rules ...validation.Rule) LoadFunc {
	return loadWithConversion(strconv.ParseBool, " is not a bool", required, rules...)
}

func loadWithConversion[T any](converter func(string) (T, error), conversionError string, required bool, rules ...validation.Rule) LoadFunc {
	return func(key string, params map[string][]string) (interface{}, error) {
		if _, exists := params[key]; !exists {
			if required {
				return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s can't be empty", key))
			} else {
				return nil, nil
			}
		}
		if len(params[key]) != 1 {
			return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s has to hold exactly one value", key))
		}
		value, err := converter(params[key][0])
		if err != nil {
			return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s%s", key, conversionError))
		}
		err = validation.Validate(value, rules...)
		if err != nil {
			return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s: %s", key, err.Error()))
		}
		return value, nil
	}
}

func LoadStringArray(required bool, rules ...validation.Rule) LoadFunc {
	return func(key string, params map[string][]string) (interface{}, error) {
		if _, exists := params[key]; !exists {
			if required {
				return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s can't be empty", key))
			} else {
				return nil, nil
			}
		}
		if len(params[key]) != 1 {
			return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s has to hold exactly one value", key))
		}
		data := params[key][0]
		var value []string
		err := json.Unmarshal([]byte(data), &value)
		if err != nil {
			return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s: could not parse string array", key))
		}
		for _, element := range value {
			err = validation.Validate(element, rules...)
			if err != nil {
				return nil, util.NewErrorResponse(nil, fmt.Sprintf("%s: %s", key, err.Error()))
			}
		}
		return value, nil
	}
}

func LoadMap(prefix string, toLoad LoadData) LoadFunc {
	return func(_ string, params map[string][]string) (interface{}, error) {
		valueMap := make(map[string]interface{})
		for key, loadFunc := range toLoad {
			value, err := loadFunc(prefix+key, params)
			if err != nil {
				return nil, err
			}
			if value != nil {
				valueMap[key] = value
			}
		}
		return valueMap, nil
	}
}

func AddDefault(defaultVal interface{}, loadFunc LoadFunc) LoadFunc {
	return func(key string, params map[string][]string) (interface{}, error) {
		value, err := loadFunc(key, params)
		if err == nil && value == nil {
			return defaultVal, nil
		}
		return value, err
	}
}
