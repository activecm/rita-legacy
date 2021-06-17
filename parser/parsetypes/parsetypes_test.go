package parsetypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewBroDataFactory(t *testing.T) {

	testCasesIn := []string{"conn", "http", "dns", "httpa", "http_a", "http_eth0", "httpasdf12345=-ASDF?", "open_conn", "ASDF"}
	testCasesOut := []BroData{&Conn{}, &HTTP{}, &DNS{}, &HTTP{}, &HTTP{}, &HTTP{}, &HTTP{}, &OpenConn{}, nil}
	for i := range testCasesIn {
		factory := NewBroDataFactory(testCasesIn[i])
		if factory == nil {
			require.Nil(t, testCasesOut[i])
		} else {
			require.Equal(t, testCasesOut[i], factory())
		}
	}
}

func TestConvertTimestamp(t *testing.T) {
	testCases := []struct {
		input    interface{}
		expected int64
	}{
		{1517336042.090842, 1517336042},
		{1517336042, 1517336042},
		{"2018-01-30T18:14:02Z", 1517336042},
		{0, 0},
		{"", 0},
		{nil, 0},
	}

	for _, testCase := range testCases {
		actual := convertTimestamp(testCase.input)
		require.Equal(t, testCase.expected, actual, "input: %v", testCase.input)
	}
}
