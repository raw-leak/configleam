package encryptor_test

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/raw-leak/configleam/internal/pkg/encryptor"
	"github.com/stretchr/testify/suite"
)

type ConfigleamSecretsSuite struct {
	suite.Suite
	encryptor *encryptor.Encryptor
	key       string
}

func TestConfigleamSecretsSuite(t *testing.T) {
	suite.Run(t, new(ConfigleamSecretsSuite))
}

func (suite *ConfigleamSecretsSuite) SetupSuite() {
	var err error

	suite.key = "01234567890123456789012345678901"
	suite.encryptor, err = encryptor.NewEncryptor(suite.key)
	if err != nil {
		suite.Assert().NoError(err)
	}
}

func (suite *ConfigleamSecretsSuite) TestEncryptDecrypt() {
	testCases := []struct {
		name               string
		plaintext          []byte
		expectedEncryptErr error
		expectedDecryptErr error
	}{
		{
			name:               "Encrypt and decrypt a simple string",
			plaintext:          []byte("hello world"),
			expectedEncryptErr: nil,
			expectedDecryptErr: nil,
		},
		{
			name: "Encrypt a complex JSON-like string",
			plaintext: []byte(`{
				"name": "John Doe",
				"age": 30,
				"address": {
					"street": "123 Main St",
					"city": "town",
					"state": "CA"
				},
				"pets": [
					{
						"name": "Fido",
						"species": "Dog"
					},
					{
						"name": "Whiskers",
						"species": "Cat"
					}
				]
			}`),
			expectedEncryptErr: nil,
			expectedDecryptErr: nil,
		},
		{
			name:               "Encrypt an empty string",
			plaintext:          []byte(""),
			expectedEncryptErr: nil,
			expectedDecryptErr: nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			// non-deterministic
			encryptedData, err := suite.encryptor.Encrypt(ctx, tc.plaintext)

			if tc.expectedEncryptErr != nil {
				suite.Require().EqualError(err, tc.expectedEncryptErr.Error())
			} else {

				suite.Require().NotEqual(tc.plaintext, encryptedData)

				// Decrypt the encrypted data
				decryptedData, err := suite.encryptor.Decrypt(ctx, encryptedData)

				if tc.expectedDecryptErr != nil {
					suite.Require().EqualError(err, tc.expectedDecryptErr.Error())
				} else {
					suite.Require().NoError(err)
					suite.Require().Equal(string(tc.plaintext), string(decryptedData))
				}

			}

			// deterministic
			encryptedData, err = suite.encryptor.EncryptDet(ctx, tc.plaintext)

			if tc.expectedEncryptErr != nil {
				suite.Require().EqualError(err, tc.expectedEncryptErr.Error())
			} else {

				suite.Require().NotEqual(tc.plaintext, encryptedData)

				// Decrypt the encrypted data
				decryptedData, err := suite.encryptor.Decrypt(ctx, encryptedData)

				if tc.expectedDecryptErr != nil {
					suite.Require().EqualError(err, tc.expectedDecryptErr.Error())
				} else {
					suite.Require().NoError(err)
					suite.Require().Equal(string(tc.plaintext), string(decryptedData))
				}

			}
		})
	}
}

func (suite *ConfigleamSecretsSuite) TestEncryptDet() {
	testCases := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "Encrypt and decrypt with same key as some plain text",
			plaintext: []byte("hello world"),
		},
		{
			name:      "Encrypt and decrypt with same key as access key",
			plaintext: []byte("cfg_FpYtVdHXH3k6ahtuXpe2bRfYBDLqXNAhIQruPBGd"),
		},
		{
			name:      "Encrypt and decrypt with JSON-like string",
			plaintext: []byte(`{"name": "John Doe", "age": 30, "city": "New York"}`),
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			ctx := context.Background()

			encrypted1, err := suite.encryptor.EncryptDet(ctx, tc.plaintext)
			suite.NoError(err)

			encrypted2, err := suite.encryptor.EncryptDet(ctx, tc.plaintext)
			suite.NoError(err)

			base64Encoded1 := base64.StdEncoding.EncodeToString(encrypted1)
			base64Encoded2 := base64.StdEncoding.EncodeToString(encrypted2)

			suite.Equal(encrypted1, encrypted2)
			suite.Equal(base64Encoded1, base64Encoded2)
		})
	}
}
