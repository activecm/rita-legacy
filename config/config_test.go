package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestStruct struct {
	InertString       string
	ExpandString      string
	ExpandStringSlice []string
	Inner             TestStructInner
}

type TestStructInner struct {
	InertString       string
	ExpandString      string
	ExpandStringSlice []string
}

func TestExpandConfig(t *testing.T) {
	inert := "DO_NOT_CHANGE"
	outerEnvVarName := "_OUTER_ENV_VAR"
	outerEnvVarValue := "OUTER_VALUE"
	innerEnvVarName := "_INNER_ENV_VAR"
	innerEnvVarValue := "INNER_VALUE"
	test := TestStruct{
		InertString:       inert,
		ExpandString:      "$" + outerEnvVarName,
		ExpandStringSlice: []string{"$" + outerEnvVarName, inert},
	}
	innerStruct := TestStructInner{
		InertString:       inert,
		ExpandString:      "$" + innerEnvVarName,
		ExpandStringSlice: []string{"$" + innerEnvVarName, inert},
	}
	test.Inner = innerStruct

	os.Setenv(outerEnvVarName, outerEnvVarValue)
	os.Setenv(innerEnvVarName, innerEnvVarValue)
	assert.Equal(t, outerEnvVarValue, os.ExpandEnv("$"+outerEnvVarName))
	assert.Equal(t, innerEnvVarValue, os.ExpandEnv("$"+innerEnvVarName))
	expandConfig(reflect.ValueOf(&test).Elem())

	assert.Equal(t, inert, test.InertString)
	assert.Equal(t, outerEnvVarValue, test.ExpandString)
	assert.Equal(t, innerEnvVarValue, test.Inner.ExpandString)
	os.Unsetenv(outerEnvVarName)
	os.Unsetenv(innerEnvVarName)
}
