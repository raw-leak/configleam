package encryptor_test

import (
	"context"
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
		})
	}
}
