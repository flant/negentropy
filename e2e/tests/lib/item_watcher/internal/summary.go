package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flant/negentropy/vault-plugins/shared/io"
)

func NewSummaryOfTopic(topic Topic, edgeTimestamp time.Time) *SummaryOfTopic {
	return &SummaryOfTopic{
		EdgeTimestamp: edgeTimestamp,
		Topic:         topic,
		Summaries:     map[Type]map[ItemKey]ItemSummary{},
	}
}

type ItemSummary struct {
	Key                        string // object/id
	IsDeleted                  bool   // true if last was gotten tombstone
	MsgCounter                 int    // how often item was at topic
	ErrWasDeletedBeforeCreated bool   // tombstone before creation
	Recreated                  bool   // item after tombstone
}

type ItemKey = string
type Type = string

type SummaryOfTopic struct {
	// todo mutex if will used not only for e2e
	Topic         Topic
	EdgeTimestamp time.Time
	Summaries     map[Type]map[ItemKey]ItemSummary
}

// ProceedMessage fill summary of topic by using msg
func (s *SummaryOfTopic) ProceedMessage(msg io.MsgDecoded) error {
	if msg.TimeStamp.Before(s.EdgeTimestamp) {
		return nil // doesn't process
	}
	objectsOfType, mapExist := s.Summaries[msg.Type]
	if !mapExist {
		objectsOfType = map[ItemKey]ItemSummary{}
	}
	key := msg.Type + "/" + msg.ID
	old, exist := objectsOfType[key]
	if exist {
		return s.processExist(objectsOfType, key, old, msg)
	}
	itemSummary := ItemSummary{
		Key:                        key,
		IsDeleted:                  msg.IsDeleted(),
		MsgCounter:                 1,
		ErrWasDeletedBeforeCreated: msg.IsDeleted(),
		Recreated:                  false,
	}

	objectsOfType[key] = itemSummary
	s.Summaries[msg.Type] = objectsOfType
	return nil
}

func (s *SummaryOfTopic) processExist(objectsOfType map[ItemKey]ItemSummary, key string, old ItemSummary, msg io.MsgDecoded) error {
	isDeletingBeforeCreating := old.IsDeleted && msg.IsDeleted()
	isRecreating := old.IsDeleted && !msg.IsDeleted()
	itemSummary := ItemSummary{
		Key:                        key,
		IsDeleted:                  msg.IsDeleted(),
		MsgCounter:                 old.MsgCounter + 1,
		ErrWasDeletedBeforeCreated: old.ErrWasDeletedBeforeCreated || isDeletingBeforeCreating,
		Recreated:                  old.Recreated || isRecreating,
	}
	objectsOfType[key] = itemSummary
	s.Summaries[msg.Type] = objectsOfType
	return nil
}

type Report struct {
	Total                   int
	NotDeleted              int
	WasDeletedBeforeCreated int
	Recreated               int
}

func (r Report) String() string {
	return fmt.Sprintf("total: %d not_deleted: %d err: %d suspicted: %d", r.Total, r.NotDeleted,
		r.WasDeletedBeforeCreated, r.Recreated)
}

func (r *Report) Add(r2 Report) {
	r.Total += r2.Total
	r.NotDeleted += r2.NotDeleted
	r.WasDeletedBeforeCreated += r2.WasDeletedBeforeCreated
	r.Recreated += r2.Recreated
}

type ReportOfType struct {
	ObjectType Type
	Report
}

func makeReportForType(objectType Type, itemsOfType map[ItemKey]ItemSummary) ReportOfType {
	result := ReportOfType{ObjectType: objectType}
	for _, object := range itemsOfType {
		result.Total += 1
		if !object.IsDeleted {
			result.NotDeleted += 1
		}
		if object.ErrWasDeletedBeforeCreated {
			result.WasDeletedBeforeCreated += 1
		}
		if object.Recreated {
			result.Recreated += 1
		}
	}
	return result
}

type ReportOfTopic struct {
	TopicName string
	Report
	TypesReports []ReportOfType
}

func makeReportForTopic(summaryOfTopic SummaryOfTopic) ReportOfTopic {
	result := ReportOfTopic{TopicName: summaryOfTopic.Topic.Name}
	for objectType, typeSummaries := range summaryOfTopic.Summaries {
		typeReport := makeReportForType(objectType, typeSummaries)
		result.Add(typeReport.Report)
		result.TypesReports = append(result.TypesReports, typeReport)
	}
	sort.Slice(result.TypesReports, func(i, j int) bool {
		return strings.Compare(result.TypesReports[i].ObjectType, result.TypesReports[j].ObjectType) > 0
	})

	return result
}

func MakeReport(summaries []*SummaryOfTopic) (Report, []ReportOfTopic) {
	result := Report{}
	var results []ReportOfTopic
	for _, summaryOfTopic := range summaries {
		topicReport := makeReportForTopic(*summaryOfTopic)
		result.Add(topicReport.Report)
		results = append(results, topicReport)
	}
	return result, results
}

func MakeStringReport(summaries []*SummaryOfTopic) string {
	report, topicReports := MakeReport(summaries)
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("system: %s\n", report.String()))
	for _, s := range topicReports {
		builder.WriteString(s.String())
	}
	return builder.String()
}

func (r *ReportOfTopic) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("%s: %s\n", r.TopicName, r.Report.String()))
	for _, typeReport := range r.TypesReports {
		builder.WriteString(fmt.Sprintf("\t%s: %s\n", typeReport.ObjectType, typeReport.String()))
	}
	return builder.String()
}
