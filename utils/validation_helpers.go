package utils

import (
	"reflect"

	errors "github.com/zgalor/weberr"
)

type ValidateRule func() error

func ValidateNilField(field interface{}, name string) ValidateRule {
	return func() error {
		if reflect.ValueOf(field).IsNil() {
			return errors.BadRequest.UserErrorf("Missing field '%s'", name)
		}
		return nil
	}
}

func ValidateNilObject(field interface{}, name string) ValidateRule {
	return func() error {
		if reflect.ValueOf(field).IsNil() {
			return errors.BadRequest.UserErrorf("'%s' is missing", name)
		}
		return nil
	}
}

func Validate(rules []ValidateRule) error {
	for _, rule := range rules {
		if err := rule(); err != nil {
			return err
		}
	}
	return nil
}

func ValidateStringFieldNotEmpty(param *string, name string) ValidateRule {
	return func() error {
		if param == nil {
			return errors.BadRequest.UserErrorf("Missing field '%s'", name)
		}
		if len(*param) == 0 {
			return errors.BadRequest.UserErrorf("Field '%s' is empty", name)
		}
		return nil
	}
}

func Contains[T comparable](slice []T, element T) bool {
	for _, sliceElement := range slice {
		if sliceElement == element {
			return true
		}
	}

	return false
}

func Transform[T any, Y any](elements []T, transformationFunc func(T) Y) []Y {
	results := make([]Y, 0)
	for _, elem := range elements {
		results = append(results, transformationFunc(elem))
	}
	return results
}
