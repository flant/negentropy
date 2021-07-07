package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/flant/negentropy/authd/pkg/log"
)

func ServeJSON(w http.ResponseWriter, r *http.Request, requestObj interface{}, h func(ctx context.Context) (interface{}, int, error)) {
	logEntry := log.GetLogger(r.Context())

	bodyBytes, err := readBody(r)
	if err != nil {
		logEntry.Debugf("read JSON body: %v", err)
		writeServerErrorf(w, "read body: %v", err)
		return
	}

	err = json.Unmarshal(bodyBytes, requestObj)
	if err != nil {
		logEntry.Debugf("unmarshal JSON from '%s': %v", string(bodyBytes), err)
		writeServerErrorf(w, "unmarshal: %v", err.Error())
		return
	}

	resp, status, err := h(r.Context())
	if err != nil {
		// Check if ClientError
		clientError, ok := err.(ClientError) // type assertion for behavior.
		if ok {
			body, bodyErr := clientError.ResponseBody()
			if bodyErr != nil {
				logEntry.Debugf("handler return vault client error: %v", err)
				writeServerErrorf(w, "vault client error: %v", err.Error())
			}
			status, headers := clientError.ResponseHeaders()
			w.WriteHeader(status)
			for k, v := range headers {
				w.Header().Set(k, v)
			}
			w.Write(body)
			return
		}

		logEntry.Debugf("call handler: %v", err)
		writeServerErrorf(w, "handle")
		return
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		logEntry.Debugf("handler return bad response: %v: '%s'", err, string(respBytes))
		writeServerErrorf(w, "bad response")
		return
	}

	logEntry.Debugf("handler run success")

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(respBytes)
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body")
	}
	return body, nil
}

func writeServerErrorf(w http.ResponseWriter, format string, args ...interface{}) {
	w.WriteHeader(http.StatusInternalServerError)
	if len(args) == 0 {
		// %!(EXTRA []interface {}=[])
		_, _ = fmt.Fprintf(w, format)
	} else {
		_, _ = fmt.Fprintf(w, format, args)
	}
}
