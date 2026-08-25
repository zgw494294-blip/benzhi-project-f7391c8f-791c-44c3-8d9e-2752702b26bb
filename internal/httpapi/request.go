package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	if media := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])); media != "application/json" {
		return &requestError{code: "UNSUPPORTED_MEDIA_TYPE", message: "Content-Type 必须为 application/json", status: 415}
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return &requestError{code: "INVALID_JSON", message: "JSON 请求体无效：" + err.Error(), status: 400}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return &requestError{code: "INVALID_JSON", message: "请求体只能包含一个 JSON 对象", status: 400}
	}
	return nil
}

type requestError struct {
	code, message string
	status        int
}

func (e *requestError) Error() string { return e.message }

func decodeOrProblem(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := decodeJSON(w, r, target); err != nil {
		issue := err.(*requestError)
		writeProblem(w, r, issue.status, issue.code, issue.message, "")
		return false
	}
	return true
}
