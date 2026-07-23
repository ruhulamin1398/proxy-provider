package ai

// SSRFError is returned when a request fails SSRF validation.
type SSRFError struct {
	Message string
}

func (e *SSRFError) Error() string {
	return e.Message
}
