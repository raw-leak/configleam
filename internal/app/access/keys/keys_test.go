package keys_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/raw-leak/configleam/internal/app/access/keys"
	"github.com/stretchr/testify/suite"
)

type KeysTestSuite struct {
	suite.Suite
	keys *keys.Keys
}

func (suite *KeysTestSuite) SetupTest() {
	suite.keys = keys.New()
}

func TestKeysTestSuite(t *testing.T) {
	suite.Run(t, new(KeysTestSuite))
}

func (suite *KeysTestSuite) TestGenerateKey() {
	key, err := suite.keys.GenerateKey(context.Background())
	suite.NoError(err, "Generating key should not produce an error")

	// Test key length
	expectedLength := len(keys.KeyPrefix) + keys.Length
	suite.Equal(expectedLength, len(key), "Generated key length is incorrect")

	// Test key prefix
	suite.Regexp(regexp.MustCompile("^"+keys.KeyPrefix), key, "Generated key does not start with the correct prefix")

	// Test key content
	charSetPattern := "[" + keys.Chars + "]+"
	suite.Regexp(regexp.MustCompile(charSetPattern), key[len(keys.KeyPrefix):], "Generated key contains invalid characters")
}

func (suite *KeysTestSuite) TestKeyUniqueness() {
	// Generate a large number of keys and ensure they are unique
	generatedKeys := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		key, err := suite.keys.GenerateKey(context.Background())
		suite.NoError(err, "Generating key should not produce an error")

		// Check for uniqueness
		_, exists := generatedKeys[key]
		suite.False(exists, "Generated key should be unique")
		generatedKeys[key] = true
	}
}
