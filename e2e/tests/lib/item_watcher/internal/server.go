package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/flant/negentropy/e2e/tests/lib/item_watcher/pkg"
	sharedkafka "github.com/flant/negentropy/vault-plugins/shared/kafka"
)

type TopicName = string

type WatcherServer struct {
	// edge timestamp - don't process messages before this timestamp
	Edge time.Time
	// topics to process
	Topics []string
	// example : ":3333"
	ListenAddress string

	initialized bool
	http.Server
	kafkaCfg        sharedkafka.BrokerConfig
	kafkaSources    []*pkg.KafkaSource
	summaries       []*SummaryOfTopic
	ShutDownRequest chan bool
}

func (s *WatcherServer) InitServer() error {
	s.kafkaCfg = sharedkafka.BrokerConfig{
		Endpoints: []string{"localhost:9094"},
		SSLConfig: &sharedkafka.SSLConfig{
			UseSSL:                true,
			CAPath:                "../docker/kafka/ca.crt",
			ClientPrivateKeyPath:  "../docker/kafka/client.key",
			ClientCertificatePath: "../docker/kafka/client.crt",
		},
	}
	err := s.buildAndRunKafkaSources()
	if err != nil {
		return fmt.Errorf("building and running kafka sources: %w", err)
	}
	s.Server = http.Server{
		Addr:         s.ListenAddress,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.Handler = s.buildRouter()
	s.ShutDownRequest = make(chan bool)
	s.initialized = true
	return nil
}

func (s *WatcherServer) RunServer() {
	if !s.initialized {
		log.Fatalf("CRITICAL ERROR: server not initialized, use 'InitServer' first")
	}
	log.Printf("Listening at http://%s", s.ListenAddress)

	if err := http.ListenAndServe(s.ListenAddress, s.Server.Handler); err != http.ErrServerClosed {
		log.Fatalf("HTTP server ListenAndServe Error: %v", err)
	}
}

func (s *WatcherServer) Shutdown(ctx context.Context) {
	s.stopKafkaSources()
	err := s.Server.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
}

func (s *WatcherServer) buildAndRunKafkaSources() error {
	groupID := "item-watcher-" + strings.ReplaceAll(time.Now().String()[11:23], ":", "-")
	mb := pkg.MessageBroker(s.kafkaCfg, groupID, hclog.Default())
	memstore := pkg.EmptyMemstore(mb, hclog.NewNullLogger())
	for _, topic := range s.Topics {
		topicSummary := NewSummaryOfTopic(topic, s.Edge)
		s.summaries = append(s.summaries, topicSummary)
		ks := pkg.NewKafkaSource(mb, memstore, topic, groupID, hclog.Default(), topicSummary)
		s.kafkaSources = append(s.kafkaSources, ks)
		go func() { ks.Run() }()
	}
	return nil
}

func (s *WatcherServer) stopKafkaSources() {
	for _, ks := range s.kafkaSources {
		ks.Stop()
	}
}

func (s *WatcherServer) shutDownHandler(writer http.ResponseWriter, _ *http.Request) {
	io.WriteString(writer, "shutting down server")
	s.ShutDownRequest <- true
}

type route struct {
	path    string
	handler func(writer http.ResponseWriter, request *http.Request)
}

func (s *WatcherServer) buildRouter() *mux.Router {
	router := mux.NewRouter()

	routes := []route{
		{"/report", s.reportHandler},
		{"/json_report", s.jsonReportHandler},
		{"/summary/{topic}", s.topicSummaryHandler},
		{"/summary/{topic}/{object_type}", s.objectTypeSummaryHandler},
		{"/summary/{topic}/{object_type}/{id}", s.objectSummaryHandler},
		{"/shutdown", s.shutDownHandler},
	}

	for _, route := range routes {
		router.HandleFunc(route.path, route.handler)
	}

	router.NotFoundHandler = Helper{routes: routes}

	return router
}

func (s *WatcherServer) reportHandler(w http.ResponseWriter, _ *http.Request) {
	report := MakeStringReport(s.summaries)
	io.WriteString(w, report)
}

func (s *WatcherServer) jsonReportHandler(writer http.ResponseWriter, _ *http.Request) {
	systemReport, topicsReports := MakeReport(s.summaries)
	result := struct {
		System Report
		Topics []ReportOfTopic
	}{
		System: systemReport,
		Topics: topicsReports,
	}
	writeJson(writer, result)
}

func (s *WatcherServer) topicSummaryHandler(writer http.ResponseWriter, request *http.Request) {
	s.findAndHandleTopicSummary(writer, request, func(writer http.ResponseWriter, request *http.Request, topicSummary SummaryOfTopic) {
		writeJson(writer, topicSummary)
	})
}

func (s *WatcherServer) objectTypeSummaryHandler(writer http.ResponseWriter, request *http.Request) {
	s.findAndHandleObjectTypeSummary(writer, request, func(writer http.ResponseWriter, request *http.Request, objectsTypeSummary map[ItemKey]ItemSummary) {
		writeJson(writer, objectsTypeSummary)
	})
}

func (s *WatcherServer) objectSummaryHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	objectID := vars["id"]
	objectType := vars["object_type"]
	topic := vars["topic"]
	s.findAndHandleObjectTypeSummary(writer, request, func(writer http.ResponseWriter, request *http.Request, objectTypeSummary map[ItemKey]ItemSummary) {
		objectSummary, exist := objectTypeSummary[objectType+"/"+objectID]
		if !exist {
			writeNotFound(writer, fmt.Sprintf("object %q of type  %q not found at topic  %q\n", objectID, objectType, topic))
			return
		}
		writeJson(writer, objectSummary)
	})
}

func (s *WatcherServer) findAndHandleTopicSummary(writer http.ResponseWriter, request *http.Request, topicSummaryHandler func(writer http.ResponseWriter, request *http.Request, topicSummary SummaryOfTopic)) {
	vars := mux.Vars(request)
	topic := vars["topic"]
	for _, topicSummary := range s.summaries {
		if topicSummary.TopicName == topic {
			topicSummaryHandler(writer, request, *topicSummary)
			return
		}
	}
	writeNotFound(writer, fmt.Sprintf("topic %s not found\n", topic))
}

func (s *WatcherServer) findAndHandleObjectTypeSummary(writer http.ResponseWriter, request *http.Request, objectTypeSummaryHandler func(writer http.ResponseWriter, request *http.Request, objectTypeSummary map[ItemKey]ItemSummary)) {
	vars := mux.Vars(request)
	objectType := vars["object_type"]
	s.findAndHandleTopicSummary(writer, request, func(writer http.ResponseWriter, request *http.Request, topicSummary SummaryOfTopic) {
		objectTypeSummary, exist := topicSummary.Summaries[objectType]
		if !exist {
			writeNotFound(writer, fmt.Sprintf("object_type %q not found at topic  %q\n", objectType, topicSummary.TopicName))
		}
		objectTypeSummaryHandler(writer, request, objectTypeSummary)
	})
}

func writeJson(writer http.ResponseWriter, data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		io.WriteString(writer, err.Error())
	}
	writer.Write(bytes)
}

func writeNotFound(writer http.ResponseWriter, message string) {
	writer.WriteHeader(http.StatusNotFound)
	io.WriteString(writer, message)
}

type Helper struct {
	routes []route
}

func (h Helper) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	builder := strings.Builder{}
	builder.WriteString("valid paths:\n")
	for _, route := range h.routes {
		builder.WriteString(route.path + "\n")
	}
	io.WriteString(writer, fmt.Sprintf(builder.String()))
}
