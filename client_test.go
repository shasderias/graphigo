package graphigo_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/shasderias/graphigo"
)

var specimenMetric = graphigo.Metric{"abc", 123.03, time.Unix(1234567890, 0)}

var specimenMetrics = []graphigo.Metric{
	{"abc", 123.03, time.Unix(1234567890, 0)},
	{"abc", "123.03", time.Unix(1234567891, 0)},
	{"abc", 123, time.Unix(1234567892, 0)},
}
var specimenMetrics2 = []graphigo.Metric{
	{"abc", 123.02, time.Unix(1234567890, 0)},
	{"abc", "123.02", time.Unix(1234567891, 0)},
	{"abc", 122, time.Unix(1234567892, 0)},
}

func TestSanity(t *testing.T) {
	server := graphigo.NewMockServer(t, "2003")

	client, err := graphigo.NewClient("localhost")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(specimenMetrics...); err != nil {
		t.Fatal(err)
	}
	client.Close()

	time.Sleep(100 * time.Millisecond)

	if server.HasErrors() {
		t.Fatal(server.Errors())
	}

	recvMetrics := server.Metrics()
	if len(recvMetrics) != len(specimenMetrics) {
		t.Fatalf("got %d metrics; want %d", len(recvMetrics), len(specimenMetrics))
	}

	if diff := cmp.Diff(specimenMetrics, recvMetrics, metricCmpOptions()...); diff != "" {
		t.Fatal(diff)
	}
	// safety guard to make sure metricCmpOptions is correct
	if diff := cmp.Diff(specimenMetrics2, recvMetrics, metricCmpOptions()...); diff == "" {
		t.Fatal("expected diff")
	}
}

func TestNonDefaultPort(t *testing.T) {
	server := graphigo.NewMockServer(t, "2004")

	client, err := graphigo.NewClient("localhost:2004")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(specimenMetric); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	if server.HasErrors() {
		t.Fatal(server.Errors())
	}
}

func TestPrefix(t *testing.T) {
	server := graphigo.NewMockServer(t, "2005")

	clientWithNoDotSuffix, err := graphigo.NewClient("localhost:2005", func(c *graphigo.Config) {
		c.Prefix = "prefix"
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := clientWithNoDotSuffix.Send(specimenMetric); err != nil {
		t.Fatal(err)
	}
	clientWithNoDotSuffix.Close()

	clientWithDotSuffix, err := graphigo.NewClient("localhost:2005", func(c *graphigo.Config) {
		c.Prefix = "prefix."
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := clientWithDotSuffix.Send(specimenMetric); err != nil {
		t.Fatal(err)
	}
	clientWithDotSuffix.Close()

	time.Sleep(100 * time.Millisecond)

	if server.HasErrors() {
		t.Fatal(server.Errors())
	}

	recvMetrics := server.Metrics()
	if len(recvMetrics) != 2 {
		t.Fatalf("got %d metrics; want %d", len(recvMetrics), 2)
	}

	if recvMetrics[0].Path != "prefix.abc" {
		t.Fatalf("got %s; want %s", recvMetrics[0].Path, "prefix.abc")
	}
	if diff := cmp.Diff(specimenMetric, recvMetrics[0],
		cmpopts.EquateApproxTime(1*time.Second), cmpopts.IgnoreFields(graphigo.Metric{}, "Path")); diff != "" {
		t.Fatal(diff)
	}
	if recvMetrics[1].Path != "prefix.abc" {
		t.Fatalf("got %s; want %s", recvMetrics[1].Path, "prefix.abc")
	}
	if diff := cmp.Diff(specimenMetric, recvMetrics[1],
		cmpopts.EquateApproxTime(1*time.Second), cmpopts.IgnoreFields(graphigo.Metric{}, "Path")); diff != "" {
		t.Fatal(diff)
	}
	if specimenMetric.Path != "abc" {
		t.Fatalf("setting prefix should not alter slice passed to Send(), got %s; want %s", specimenMetric.Path, "abc")
	}
}

func TestEmpty(t *testing.T) {
	server := graphigo.NewMockServer(t, "2006")

	client, err := graphigo.NewClient("localhost:2006")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(); err != nil {
		t.Fatal(err)
	}
	client.Close()

	time.Sleep(100 * time.Millisecond)

	if server.HasErrors() {
		t.Fatal(server.Errors())
	}
	if metrics := server.Metrics(); len(metrics) > 0 {
		t.Fatal("expected no metrics")
	}
}

func TestInvalidMetric(t *testing.T) {
	_ = graphigo.NewMockServer(t, "2007")

	client, err := graphigo.NewClient("localhost:2007")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	if err := client.Send(graphigo.Metric{}); err == nil {
		t.Fatal("expected error")
	}
	if err := client.Send(graphigo.Metric{Value: 3.14, Timestamp: time.Now()}); err == nil {
		t.Fatal("expected error")
	}
	if err := client.Send(graphigo.Metric{Path: "apple", Value: 3.14}); err == nil {
		t.Fatal("expected error")
	}
}

func TestSendAfterClose(t *testing.T) {
	server := graphigo.NewMockServer(t, "2008")

	client, err := graphigo.NewClient("localhost:2008")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Send(specimenMetric); err != nil {
		t.Fatal(err)
	}
	client.Close()
	if err := client.Send(specimenMetric); err != nil {
		t.Fatal(err)
	}
	client.Close()

	time.Sleep(100 * time.Millisecond)

	if server.HasErrors() {
		t.Fatal(server.Errors())
	}

	metrics := server.Metrics()
	if len(metrics) != 2 {
		t.Fatalf("got %d; want %d", len(metrics), 2)
	}
	if diff := cmp.Diff(specimenMetric, metrics[0], metricCmpOptions()...); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(specimenMetric, metrics[1], metricCmpOptions()...); diff != "" {
		t.Fatal(diff)
	}
}

func metricCmpOptions() []cmp.Option {
	return []cmp.Option{
		cmpopts.EquateApproxTime(time.Second),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p[len(p)-1].String() == ".Value"
		}, cmp.Transformer("Metric.Value", func(v any) float64 {
			switch tv := v.(type) {
			case int:
				return float64(tv)
			case float64:
				return tv
			case string:
				f, err := strconv.ParseFloat(tv, 64)
				if err != nil {
					panic(err)
				}
				return f
			default:
				f, err := strconv.ParseFloat(fmt.Sprintf("%v", tv), 64)
				if err != nil {
					panic(err)
				}
				return f
			}
		})),
	}
}
