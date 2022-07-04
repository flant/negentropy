package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func main() {
	http.HandleFunc("/asdf", objectHandler)
	println("server is started, endpoint: http://localhost:9200/asdf, will process 100 requests")
	err := http.ListenAndServe(":9200", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("server closed\n")
	} else if err != nil {
		fmt.Printf("error starting server: %s\n", err)
		os.Exit(1)
	}
}

var counter = 0
var errCounter = 0

func objectHandler(rw http.ResponseWriter, req *http.Request) {
	if counter > 100 {
		if errCounter == 0 {
			os.Exit(0)
		}
		println("err count:", errCounter)
		os.Exit(1)
	}
	counter += 1
	err := processRequest(rw, req)
	if err != nil {
		errCounter += 1
		println("error:", err.Error())
		rw.WriteHeader(400)
		rw.Write([]byte(err.Error())) // nolint:errcheck
	} else {
		println("ok, counter =", counter)
		rw.WriteHeader(200)
	}
}

func processRequest(rw http.ResponseWriter, req *http.Request) error {
	if req.Method != "POST" {
		return fmt.Errorf("wrong method:%s", req.Method) // nolint:errcheck
	}

	body := req.Body
	defer body.Close()
	var data map[string]interface{}

	err := json.NewDecoder(req.Body).Decode(&data)
	if err != nil {
		return fmt.Errorf("getting body:%s", err) // nolint:errcheck
	}
	if data["key"] == nil {
		return fmt.Errorf("no key in request") // nolint:errcheck
	}
	if data["object"] == nil {
		return fmt.Errorf("no object in request") // nolint:errcheck
	}
	key, ok := data["key"].(string)
	if !ok {
		return fmt.Errorf(fmt.Sprintf("key: %#v should be string", data["key"])) // nolint:errcheck
	}

	object, ok := data["object"].(string)
	if !ok {
		return fmt.Errorf(fmt.Sprintf("object: %#v should be string", data["object"])) // nolint:errcheck
	}
	splitted := strings.Split(key, "/")
	if len(splitted) != 2 {
		return fmt.Errorf("wrong format of key: %s", key) // nolint:errcheck
	}

	if !strings.Contains(object, splitted[1]) {
		return fmt.Errorf("object: %s should contains key: %s", object, splitted[1]) // nolint:errcheck
	}
	return nil
}
