package e2e

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/flant/negentropy/e2e/tests/lib"
	"github.com/flant/negentropy/vault-plugins/flant_iam/backend/tests/specs"
	"github.com/flant/negentropy/vault-plugins/flant_iam/model"
)

func Test_Consumer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Consumer read kafka and put to http-gateway")
}

var tenants []model.Tenant
var messages = 10
var endChan = make(chan struct{})

var _ = BeforeSuite(func() {
	rootClient := lib.NewConfiguredIamVaultClient()
	tenantAPI := lib.NewTenantAPI(rootClient)
	for i := 0; i < messages; i++ {
		tenant := specs.CreateRandomTenant(tenantAPI)
		tenants = append(tenants, tenant)
	}
}, 1.0)

var _ = It("objectHandler should got all of messages", func() {
	go func() {
		http.HandleFunc("/foobar", objectHandler)
		fmt.Printf("server is started, endpoint: http://localhost:9200/foobar, will process %d requests\n", messages)
		err := http.ListenAndServe(":9200", nil)
		if errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("server closed\n")
		} else if err != nil {
			fmt.Printf("error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	go func() {
		time.Sleep(time.Second * 30)
		endChan <- struct{}{}
		fmt.Println("exit by timeout")
	}()

	<-endChan
	Expect(counter).To(Equal(messages), "should count all messages")
	Expect(errCounter).To(Equal(0), "shoud not be errors")
})

var counter = 0
var errCounter = 0

func objectHandler(rw http.ResponseWriter, req *http.Request) {
	if counter < messages {
		counter += 1
	}
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
	if counter == messages {
		endChan <- struct{}{}
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
