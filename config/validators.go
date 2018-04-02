package config

import (
	"reflect"
	"fmt"
)


// ValidationError
type ValidationError struct {
	msg string
	value interface{}
}

func (v ValidationError) Error() string {
	return v.msg
}


//
type ValidatorFunc func(interface{}) error

// KindValidator checks value is of a certain kind
func KindValidator(kind reflect.Kind) ValidatorFunc {
	
	validator := func(val interface{}) error {
		if reflect.TypeOf(val).Kind() != kind {
			msg := fmt.Sprintf("%v (%v) type is not %v", val, reflect.TypeOf(val).Kind(), kind)
			return ValidationError{msg:msg, value: val}
		}
		return nil
	}

	return validator
}

// TypeOfSliceValidator
func SliceOfValidator(kind reflect.Kind) ValidatorFunc {
	
	validator := func(val interface{}) error {

		// Check slice elements kind
		if reflect.TypeOf(val).Elem().Kind() != kind {
			msg := fmt.Sprintf("Invalid slice type %v expecting %v", 
				reflect.TypeOf(val).Elem().Kind(), kind)
			return ValidationError{msg:msg, value: val}
		}

		// Passed
		return nil
	}

	return ChainValidator(KindValidator(reflect.Slice), validator)
}

// StringSliceValidator
func StringSliceValidator() ValidatorFunc {
	return SliceOfValidator(reflect.String)
}

// SliceElemeValidator applies a validator function to all the elements of an slice
// the validation fails if any of the individual element validations fail.
func SliceElemValidator(val ValidatorFunc) ValidatorFunc {
	
	validator := func(value interface{}) error {
		for _, elem := range value.([]interface{}) {
			if err := val(elem); err != nil {
				return err
			}
		}

		return nil
	}

	return ChainValidator(KindValidator(reflect.Slice), validator)
}

// IntegerValidator
func IntegerValidator() ValidatorFunc {
	return KindValidator(reflect.Int64)
}

// IntegerMaxValidator checks value is smaller or equal to the max (assumes it is an int64)
func IntegerMaxValidator(max int64) ValidatorFunc {
	
	validator := func(val interface{}) error {

		// Check max
		if val.(int64) > max {
			msg := fmt.Sprintf("%v is greater than the max %v", val, max)
			return ValidationError{msg: msg, value: val}
		}

		// Passed
		return nil
	}

	return ChainValidator(IntegerValidator(), validator)
}

// IntegerMinValidator check values is greater or equal to the min (assume it is an int64)
func IntegerMinValidator(min int64) ValidatorFunc {
	
	validator := func(val interface{}) error {

		// Check min
		if val.(int64) < min {
			msg := fmt.Sprintf("%v is smalled than them min %v", val, min)
			return ValidationError{msg: msg, value: val}
		}

		// Passed
		return nil
	}

	return ChainValidator(IntegerValidator(), validator)
}

// IntegerMinMaxValidator 
func IntegerMinMaxValidator(min, max int64) ValidatorFunc {
	return ChainValidator(
		IntegerValidator(), 
		IntegerMinValidator(min), 
		IntegerMaxValidator(max))
}

// Uint16Validator
func Uint16Validator() ValidatorFunc {
	return IntegerMinMaxValidator(0, 65535)
}

// StringValidator
func StringValidator() ValidatorFunc {
	return KindValidator(reflect.String)
}

// StringMaxLengthValidator
func StringMaxLengthValidator(maxLength int) ValidatorFunc {

	validator := func(val interface{}) error {
		if len(val.(string)) > maxLength {
			msg := fmt.Sprintf("string is longer than the %v max", maxLength)
			return ValidationError{msg: msg, value: val}
		}
		return nil
	}

	return ChainValidator(StringValidator(), validator)
}

// StringMinLengthValidator
func StringMinLengthValidator(minLength int) ValidatorFunc {

	validator := func(val interface{}) error {
		if len(val.(string)) < minLength {
			msg := fmt.Sprintf("string is shorter than the %v min", minLength)
			return ValidationError{msg: msg, value: val}
		}
		return nil
	}

	return ChainValidator(StringValidator(), validator)

}

// BoolValidator
func BoolValidator() ValidatorFunc {
	return KindValidator(reflect.Bool)
}

// ChainValidators apply a list of validators, the value is valid if all the 
// validator pass
func ChainValidator(validators ...ValidatorFunc) ValidatorFunc {

	validator := func(val interface{}) error {
		for _, validator := range validators {
			if err := validator(val); err != nil {
				return err
			}
		}
		return nil
	}

	return validator
}


// StringChoiceValidator only accepts strings equal to one of the choices
func StringChoiceValidator(choices ...string) ValidatorFunc {

	// Create map with all valid strings
	validMap := make(map[string]bool)
	for _, choice := range choices {
		validMap[choice] = true
	}

	//
	validator := func(val interface{}) error {
		if _, ok := validMap[val.(string)]; !ok {
			msg := fmt.Sprintf("%v is not one of the allowed choices %v", choices)
			return  ValidationError{msg: msg, value: val}
		}

		return nil
	}

	return ChainValidator(StringValidator(), validator)
}


