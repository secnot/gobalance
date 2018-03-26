package config


import (
	"testing"
)


// Check default option values pass validation
func TestOptionsDefaults(t *testing.T) {

	for _, option := range Options {

		// Validate default
		if err := option.val(option.def); err != nil {
			t.Error(err)
		}
	}
}
