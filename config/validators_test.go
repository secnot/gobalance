package config


import(
	"testing"
	"reflect"
)


// Test KindValidator
func TestKindValidator(t *testing.T) {

	stringValidator := KindValidator(reflect.String)
	int64Validator  := KindValidator(reflect.Int64)
	uint64Validator := KindValidator(reflect.Uint64)
	sliceValidator  := KindValidator(reflect.Slice)

	// Pass validation
	if err := stringValidator("asdf"); err != nil {
		t.Error(err)
	}

	if err := int64Validator(int64(11)); err != nil {
		t.Error(err)
	}

	if err := uint64Validator(uint64(44)); err != nil {
		t.Error(err)
	}

	slice1 := []string {"as", "ff"}
	slice2 := []string {}

	if err := sliceValidator(slice1); err != nil {
		t.Error(err)
	}

	if err := sliceValidator(slice2); err != nil {
		t.Error(err)
	}

	// Fail validation
	if stringValidator(12) == nil {
		t.Error("KindValidator(reflect.String): Accepted an integer")
	}

	if stringValidator(int64(-33)) == nil {
		t.Error("KindValidator(reflect.String): Accepted an integer")
	}

	if int64Validator("this is a string") == nil {
		t.Error("KindValidator(reflect.Int64): Accepted a string")
	}

	if uint64Validator(int64(22)) == nil {
		t.Error("KindValidator(reflect.Uint64): Accepted a int64")
	}

	if sliceValidator(int64(55)) == nil {
		t.Error("KindValidator(reflect.Slice): Accepeted a int64")
	}
}


// Test SliceOfValidator
func TestSliceOfValidator(t *testing.T) {

	int64SliceValidator  := SliceOfValidator(reflect.Int64)
	stringSliceValidator := SliceOfValidator(reflect.String)

	sliceInt64       := []int64 {33, 44}
	sliceString      := []string {"this", "is", "a", "string"}
	emptySliceString := []string {}

	// Pass validation
	if err := int64SliceValidator(sliceInt64); err != nil {
		t.Error(err)
	}

	if err := stringSliceValidator(sliceString);  err != nil {
		t.Error(err)
	}

	if err := stringSliceValidator(emptySliceString); err != nil {
		t.Error(err)
	}

	// Fail validation
	if int64SliceValidator(sliceString) == nil {
		t.Error("SliceOfValidator(reflect.int64): Accepted a string slice")
	}

	if stringSliceValidator(sliceInt64) == nil {
		t.Error("SliceOfValidator(reflect.String): Accepted a int64 slice")
	}
}

// Test SliceElemValidator
func TestSliceElemValidator(t *testing.T) {
	elemValidator := Uint16Validator()
	sliceValidator := SliceElemValidator(elemValidator)

	passSlice1 := []interface{}{int64(10), int64(10), int64(50)}
	passSlice2 := []interface{}{}
	failSlice1 := []interface{}{int64(5), int64(5), int64(-10)}
	failSlice2 := []interface{}{"a one"}

	// Pass validation
	if err := sliceValidator(passSlice1); err != nil {
		t.Error(err)
	}
	
	if err := sliceValidator(passSlice2); err != nil {
		t.Error(err)
	}

	// Fail validation
	if sliceValidator(failSlice1) == nil {
		t.Error("SliceElemValidator(): Accepted invalid value")
	}

	if sliceValidator(failSlice2) == nil {
		t.Error("SliceElemeValidator(): Accepted invalid value")
	}
}


// Test IntegerValidator
func TestIntegerValidator(t *testing.T) {	
	validator := IntegerValidator()
	
	// Pass validation
	if err := validator(int64(-12)); err != nil {
		t.Error(err)
	}

	// Fail Validation
	if validator("asdf") == nil {
		t.Error("IntegerValidator(): Accepted a string")
	}

	if validator(uint32(33)) == nil {
		t.Error("InteferValidator(): Accepted a uint32")
	}
}

// Test StringValidator
func TestStringValidator(t *testing.T) {	
	
	validator := StringValidator()
	
	// Pass validation
	if err := validator("asdf"); err != nil {
		t.Error(err)
	}
	if err := validator(""); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator(32) == nil {
		t.Error("StringValidator(): Accepted a int")
	}

	if validator(uint32(33)) == nil {
		t.Error("StringValidator(): Accepted a uint32")
	}

	if validator(false) == nil {
		t.Error("StringValidator(): Accepted a bool")
	}
}

// Test BoolValidator
func TestBoolValidator(t *testing.T) {	

	validator := BoolValidator()
	
	// Pass validation
	if err := validator(true); err != nil {
		t.Error(err)
	}
	if err := validator(false); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator(32) == nil {
		t.Error("BoolValidator(): Accepted a int")
	}

	if validator(uint32(33)) == nil {
		t.Error("BoolValidator(): Accepted a uint32")
	}

	if validator("sdfsdf") == nil {
		t.Error("BoolValidator(): Accepted a string")
	}

}

// Test IntegerMaxValidator
func TestIntegerMaxValidator(t *testing.T) {

	validator := IntegerMaxValidator(44)

	// Pass validation
	if err := validator(int64(33)); err != nil {
		t.Error(err)
	}

	// Fail validation 
	if validator(int64(100)) == nil {
		t.Error("IntegerMaxValidator(44): Accepted 100")
	}

	if validator("a string") == nil {
		t.Error("IntegerMaxValidator(44): Accepted a string")
	}
}

// Test IntegerMinValidator
func TestIntegerMinValidator(t *testing.T) {

	validator := IntegerMinValidator(88)

	// Pass validation
	if err := validator(int64(100)); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator(int64(-12)) == nil {
		t.Error("IntegerMinValidator(88): Accepted -12")
	}

	if validator("a string") == nil {
		t.Error("IntegerMinValidator(88): Accepted a string")
	}

}

// Test IntegerMinMaxValidator
func TestIntegerMinMaxValidator(t *testing.T) {
	validator := IntegerMinMaxValidator(10, 100)

	// Pass validation
	if err := validator(int64(50)); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator(int64(1)) == nil {
		t.Error("IntegerMinMaxValidator(10, 100): Accepted 1")
	}

	if validator(int64(150)) == nil {
		t.Error("IntegerMinMaxValidator(10, 100): Accepted 150")
	}
}

// Test Uint16Validatori
func TestUint16Validator(t *testing.T) {
	validator := Uint16Validator()

	// Pass Validation
	if err := validator(int64(0)); err != nil {
		t.Error(err)
	}

	if err := validator(int64(65535)); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator(int64(-1)) == nil {
		t.Error("Uint16Validator: Accepted out of range integer")
	}

	if validator(int64(10000000)) == nil {
		t.Error("Uint16Validator(): Accepted out of range integer")
	}
}

// Test StringChoiceValidator
func TestStringChoiceValidator(t *testing.T) {
	validator := StringChoiceValidator("one", "two", "three")

	// Pass validation
	if err := validator("one"); err != nil {
		t.Error(err)
	}

	// Fail validation
	if validator("other string") == nil {
		t.Error("StringChoiceValidator(): Accepted a string not in the allowed choices")
	}

	if validator(uint64(323)) == nil {
		t.Error("StringChoiceValidator(): Accepted an integer")
	}


}
