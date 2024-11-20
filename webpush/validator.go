package webpush

import (
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/go-playground/validator/v10"
)

const (
	EPOCH_GT_NOW = "epoch-gt-now"
	MAILTO       = "mailto"
	ORIGIN       = "origin"
)

var _customValidator *validator.Validate

func CustomValidateStruct(s any) (err error) {
	validate := NewCustomValidator()

	if err = validate.Struct(s); err != nil {
		if _, ok := err.(*validator.InvalidValidationError); ok {
			log.Println(err)
			return
		}

		for _, err := range err.(validator.ValidationErrors) {
			log.Printf("[%s] %s = %v\n", err.StructField(), err.Tag(), err.Value())

			return fmt.Errorf("[%s] invalid value \"%s\" for tag: %s", err.StructField(), err.Value(), err.Tag())
		}
	}

	return
}

func NewCustomValidator() *validator.Validate {
	if _customValidator != nil {
		return _customValidator
	}

	_customValidator = validator.New(validator.WithRequiredStructEnabled())

	customValidators := []struct {
		tag string
		cb  validator.Func
	}{
		{EPOCH_GT_NOW, validateEpochGreaterNow},
		{MAILTO, validateMailto},
		{ORIGIN, validateOrigin},
	}

	for _, vv := range customValidators {
		if err := _customValidator.RegisterValidation(vv.tag, vv.cb); err != nil {
			log.Println(err)
			log.Fatalf("adding custom validator for %s failed", vv.tag)
		}
	}

	return _customValidator
}

func validateEpochGreaterNow(fl validator.FieldLevel) bool {
	var epoch int64

	now := time.Now().Unix()

	switch t := fl.Field().Interface().(type) {
	case int64:
		epoch = t
	case Epoch:
		epoch = t.Unix()
	case EpochMillis:
		epoch = t.Unix()
	case time.Time:
		epoch = t.Unix()
	default:
		return false
	}

	return epoch >= now
}

func validateMailto(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)

	if !ok {
		return ok
	}

	regex := `^mailto:[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$`
	r, _ := regexp.Compile(regex)

	return r.MatchString(val)
}

func validateOrigin(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)

	if !ok {
		return ok
	}

	regex := `^https?:\/\/(?:[a-zA-Z0-9-]+\.)+[a-zA-Z]{2,6}(?::\d{1,5})?$|^https?:\/\/(?:\d{1,3}\.){3}\d{1,3}(?::\d{1,5})?$`
	r, _ := regexp.Compile(regex)

	return r.MatchString(val)
}
