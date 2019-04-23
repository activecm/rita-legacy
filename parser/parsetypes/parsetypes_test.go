package parsetypes

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBroDataFactory(t *testing.T) {

	testCasesIn := []string{"conn", "http", "dns", "httpa", "http_a", "http_eth0", "httpasdf12345=-ASDF?", "ASDF"}
	testCasesOut := []BroData{&Conn{}, &HTTP{}, &DNS{}, &HTTP{}, &HTTP{}, &HTTP{}, &HTTP{}, nil}
	for i := range testCasesIn {
		factory := NewBroDataFactory(testCasesIn[i])
		if factory == nil {
			require.Nil(t, testCasesOut[i])
		} else {
			require.Equal(t, testCasesOut[i], factory())
		}
	}
}
