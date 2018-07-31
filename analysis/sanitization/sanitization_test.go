package sanitization

import (
	"testing"

	"github.com/activecm/rita/parser/parsetypes"
	"github.com/stretchr/testify/assert"
	"github.com/globalsign/mgo/bson"
)

func TestSanitizeHTTPRecord(t *testing.T) {
	hostPrefix := parsetypes.HTTP{
		Host: "1.1.1.1",
		URI:  "1.1.1.1:443",
	}
	hostPrefixSanitizedURI := ""
	t.Run("Host Prefix of URI", func(t *testing.T) {
		update := sanitizeHTTPRecord(&hostPrefix)
		newURI := extractURIUpdate(t, update)
		assert.Equal(t, hostPrefixSanitizedURI, newURI)
	})

	absoluteURI := parsetypes.HTTP{
		Host: "www.google.com",
		URI:  "http://www.google.com",
	}
	relativeURI := "/"
	t.Run("URI Absolute", func(t *testing.T) {
		update := sanitizeHTTPRecord(&absoluteURI)
		newURI := extractURIUpdate(t, update)
		assert.Equal(t, relativeURI, newURI)
	})

	goodURI := parsetypes.HTTP{
		Host: "www.google.com",
		URI:  "/images",
	}
	t.Run("Good URI", func(t *testing.T) {
		update := sanitizeHTTPRecord(&goodURI)
		assert.Nil(t, update)
	})
}

func extractURIUpdate(t *testing.T, updateDoc bson.M) string {
	assert.NotNil(t, updateDoc)
	documentInterface, ok := updateDoc["$set"]
	assert.True(t, ok)
	document, ok := documentInterface.(bson.D)
	assert.True(t, ok)
	str, ok := document[0].Value.(string)
	assert.True(t, ok)
	return str
}
