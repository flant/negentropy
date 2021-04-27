package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

func ServeJSON(w http.ResponseWriter, r *http.Request, requestObj interface{}, h func() (interface{}, int, error)) {
	bodyBytes, err := readBody(r)
	if err != nil {
		writeServerErrorf(w, "read body", err)
		return
	}

	err = json.Unmarshal(bodyBytes, requestObj)
	if err != nil {
		writeServerErrorf(w, "unmarshal: %v", err.Error())
		return
	}

	resp, status, err := h()
	if err != nil {
		// Check if ClientError
		clientError, ok := err.(ClientError) // type assertion for behavior.
		if ok {
			body, bodyErr := clientError.ResponseBody()
			if bodyErr != nil {
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

		// TODO use log
		fmt.Printf("handle: %v\n", err)
		writeServerErrorf(w, "handle")
		return
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		// TODO use log
		fmt.Printf("bad response: %v\n", err)
		writeServerErrorf(w, "bad response")
		return
	}

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
