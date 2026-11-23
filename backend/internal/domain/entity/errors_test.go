package entity

import (
	"testing"

	apperrors "github.com/eskermese/machinery-catalog/backend/pkg/apperrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSharedErrorHelpers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      error
		expected apperrors.DomainError
	}{
		{
			name: "sharedInternalError",
			err:  sharedInternalError(),
			expected: apperrors.DomainError{
				ErrorType:   apperrors.InternalErrorType,
				Code:        apperrors.InternalCode,
				Description: "внутренняя ошибка сервиса",
				Retryable:   true,
			},
		},
		{
			name: "sharedFormDecodeError",
			err:  sharedFormDecodeError(),
			expected: apperrors.DomainError{
				ErrorType:   apperrors.InvalidArgumentErrorType,
				Code:        apperrors.FormDecodeCode,
				Description: "не удалось декодировать входной запрос",
				Retryable:   false,
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var domainErr *apperrors.DomainError
			require.ErrorAs(t, c.err, &domainErr)

			assert.Equal(t, c.expected, *domainErr)
		})
	}
}

func TestDomainErrorCodes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		err          error
		expectedCode string
	}{
		{
			name:         "ErrInternal",
			err:          ErrInternal,
			expectedCode: apperrors.InternalCode,
		},
		{
			name:         "ErrFormDecode",
			err:          ErrFormDecode,
			expectedCode: apperrors.FormDecodeCode,
		},
		{
			name:         "ErrNilPointer",
			err:          ErrNilPointer,
			expectedCode: CodePrefix + "INTERNAL",
		},
		{
			name:         "ErrDatabaseCommon",
			err:          ErrDatabaseCommon,
			expectedCode: CodePrefix + "INTERNAL",
		},
		{
			name:         "ErrVersionConflict",
			err:          ErrVersionConflict,
			expectedCode: CodePrefix + "VERSION_CONFLICT",
		},
		{
			name:         "ErrInvalidID",
			err:          ErrInvalidID,
			expectedCode: CodePrefix + "INVALID_ID",
		},
		{
			name:         "ErrAuthRequired",
			err:          ErrAuthRequired,
			expectedCode: CodePrefix + "AUTH_REQUIRED",
		},
		{
			name:         "ErrRoleForbidden",
			err:          ErrRoleForbidden,
			expectedCode: CodePrefix + "ROLE_FORBIDDEN",
		},
		{
			name:         "ErrTokenExpired",
			err:          ErrTokenExpired,
			expectedCode: CodePrefix + "TOKEN_EXPIRED",
		},
		{
			name:         "ErrUserNotFound",
			err:          ErrUserNotFound,
			expectedCode: CodePrefix + "USER_NOT_FOUND",
		},
		{
			name:         "ErrPhoneInvalid",
			err:          ErrPhoneInvalid,
			expectedCode: CodePrefix + "PHONE_INVALID",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.name, func(t *testing.T) {
			t.Parallel()

			var domainErr *apperrors.DomainError
			require.ErrorAs(t, c.err, &domainErr)

			assert.Equal(t, c.expectedCode, domainErr.Code)
		})
	}
}
