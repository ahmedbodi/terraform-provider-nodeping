package client

import "testing"

func TestAPIErrorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *APIError
		want string
	}{
		{
			name: "without request id",
			err:  &APIError{StatusCode: 500, Message: "boom"},
			want: "NodePing API error (status 500): boom",
		},
		{
			name: "with request id",
			err:  &APIError{StatusCode: 400, Message: "bad", RequestID: "req-1"},
			want: "NodePing API error (status 400, request req-1): bad",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIErrorClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status        int
		isNotFound    bool
		isUnauthentic bool
		isRetryable   bool
	}{
		{status: 200, isNotFound: false, isUnauthentic: false, isRetryable: false},
		{status: 400, isNotFound: false, isUnauthentic: false, isRetryable: false},
		{status: 401, isNotFound: false, isUnauthentic: true, isRetryable: false},
		{status: 403, isNotFound: false, isUnauthentic: true, isRetryable: false},
		{status: 404, isNotFound: true, isUnauthentic: false, isRetryable: false},
		{status: 429, isNotFound: false, isUnauthentic: false, isRetryable: true},
		// 500 and above are retryable; 499 must not be.
		{status: 499, isNotFound: false, isUnauthentic: false, isRetryable: false},
		{status: 500, isNotFound: false, isUnauthentic: false, isRetryable: true},
		{status: 503, isNotFound: false, isUnauthentic: false, isRetryable: true},
	}

	for _, tt := range tests {
		err := &APIError{StatusCode: tt.status}
		if got := err.IsNotFound(); got != tt.isNotFound {
			t.Errorf("status %d: IsNotFound() = %v, want %v", tt.status, got, tt.isNotFound)
		}
		if got := err.IsUnauthorized(); got != tt.isUnauthentic {
			t.Errorf("status %d: IsUnauthorized() = %v, want %v", tt.status, got, tt.isUnauthentic)
		}
		if got := err.IsRetryable(); got != tt.isRetryable {
			t.Errorf("status %d: IsRetryable() = %v, want %v", tt.status, got, tt.isRetryable)
		}
	}
}

func TestNotFoundErrorError(t *testing.T) {
	t.Parallel()

	err := &NotFoundError{ResourceType: "check", ResourceID: "abc-123"}
	want := `check with ID "abc-123" not found`

	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestValidationErrorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *ValidationError
		want string
	}{
		{
			name: "with field",
			err:  &ValidationError{Field: "target", Message: "must be a URL"},
			want: `validation error for field "target": must be a URL`,
		},
		{
			name: "without field",
			err:  &ValidationError{Message: "generic failure"},
			want: "validation error: generic failure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRateLimitErrorError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *RateLimitError
		want string
	}{
		{
			name: "with retry after",
			err:  &RateLimitError{RetryAfter: 30},
			want: "rate limit exceeded, retry after 30 seconds",
		},
		{
			name: "without retry after",
			err:  &RateLimitError{},
			want: "rate limit exceeded",
		},
		{
			name: "negative retry after falls back to plain message",
			err:  &RateLimitError{RetryAfter: -1},
			want: "rate limit exceeded",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// All custom error types must satisfy the error interface.
func TestErrorTypesImplementError(t *testing.T) {
	t.Parallel()

	var _ error = &APIError{}
	var _ error = &NotFoundError{}
	var _ error = &ValidationError{}
	var _ error = &RateLimitError{}
}
