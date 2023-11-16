package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"
)

func TestL2TransactionDecode(t *testing.T) {
	type testCase struct {
		name           string
		input          []byte
		expectedResult L2Transaction
		expectedError  error
	}
	testCases := []testCase{
		{
			name:  "Happy path",
			input: []byte{128, 1, 5, 0, 0, 0, 1, 2, 3, 4, 5},
			expectedResult: L2Transaction{
				EffectiveGasPricePercentage: 128,
				IsValid:                     1,
				EncodedLength:               5,
				Encoded:                     []byte{1, 2, 3, 4, 5},
			},
			expectedError: nil,
		},
		{
			name:           "Invalid byte array length",
			input:          []byte{20, 21, 22, 23, 24},
			expectedResult: L2Transaction{},
			expectedError:  fmt.Errorf("expected minimum data length: 6, got: 5"),
		},
		{
			name: "Invalid encoded array length",

			input:          []byte{128, 1, 5, 0, 0, 0, 1, 2, 3, 4},
			expectedResult: L2Transaction{},
			expectedError:  fmt.Errorf("expected encoded length: 5, got: 4"),
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			decodedL2Transaction, err := DecodeL2Transaction(testCase.input)
			require.Equal(t, testCase.expectedError, err)
			assert.DeepEqual(t, testCase.expectedResult, *decodedL2Transaction)
		})
	}
}