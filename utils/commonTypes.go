package utils

type ErrorResponse struct {
	StatusCode    int    `json:"status_code"`
	ErrorMsg      string `json:"error_msg"`
	FnName        string `json:"fn_name,omitempty"`
	TrustVerified *bool  `json:"trust_verified,omitempty"`
}
