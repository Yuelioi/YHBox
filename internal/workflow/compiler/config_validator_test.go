package compiler

import "github.com/yottaapp/yotta/internal/configvalidator"

func testConfigValidators() configvalidator.Registry {
	registry, err := configvalidator.Seal(nil)
	if err != nil {
		panic(err)
	}
	return registry
}
