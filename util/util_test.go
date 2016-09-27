package util

import "testing"

func TestIsURL(t *testing.T) {
	urlMap := make(map[string]bool)
	urlMap["www.test.com"] = true
	urlMap["test.com"] = true
	urlMap["10.10.10.10"] = false
	urlMap["192.168.0.1"] = false
	urlMap["1"] = false
	urlMap[""] = false
	urlMap["www. space.com"] = false
	urlMap["http://www.test.com"] = true
	urlMap[longurl] = true

	fail := false
	for k, v := range urlMap {
		if IsURL(k) != v {
			t.Error("URL Failed: \"", k, "\" returned: ", !v)
			fail = true
		}
	}
	if fail {
		t.Fail()
	}
}
