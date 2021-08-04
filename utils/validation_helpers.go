package utils

import (
	"reflect"
	"strings"

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

func ValidateStringParameterNotEmpty(param *string, name string) ValidateRule {
	return func() error {
		if param == nil {
			return errors.BadRequest.UserErrorf("Missing field '%s'", name)
		}
		if ValidateNilField(param, name) != nil {
			if strings.ReplaceAll(*param, " ", "") == "" {
				return errors.BadRequest.UserErrorf("Field '%s' is empty", name)
			}
		}
		return nil
	}
}
