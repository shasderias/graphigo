package graphigo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

type MockServer struct {
	l net.Listener

	errorsMut sync.Mutex
	errors    []error

	metricsMut sync.Mutex
	metrics    []Metric
}

func NewMockServer(t *testing.T, port string) *MockServer {
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		t.Fatal(err)
	}
	server := &MockServer{l: l}
	go server.listen(t)
	t.Cleanup(func() {
		server.close()
	})
	time.Sleep(50 * time.Millisecond)
	return server
}

func (s *MockServer) listen(t *testing.T) {
	for {
		conn, err := s.l.Accept()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return
		} else if err != nil {
			t.Fatal(err)
		}

		go func(c net.Conn) (err error) {
			defer func() {
				if err != nil {
					s.appendError(err)
				}
			}()
			defer c.Close()

			bufReader := bufio.NewReader(c)

			for {
				line, err := bufReader.ReadBytes('\n')
				switch {
				case errors.Is(err, net.ErrClosed):
					return nil
				case errors.Is(err, io.EOF):
					return nil
				case err != nil:
					return fmt.Errorf("error reading line: %s, %w", line, err)
				}

				line = line[:len(line)-1] // remove newline

				data := bytes.Split(line, []byte(" "))
				if len(data) != 3 {
					return fmt.Errorf("invalid data: %s -> %v", line, data)
				}
				path, valueBytes, timestampBytes := string(data[0]), data[1], data[2]

				value, err := strconv.ParseFloat(string(valueBytes), 64)
				if err != nil {
					return fmt.Errorf("invalid value: %s", valueBytes)
				}

				timestampInt, err := strconv.ParseInt(string(timestampBytes), 10, 64)
				if err != nil {
					return fmt.Errorf("invalid timestamp: %s", timestampBytes)
				}
				timestamp := time.Unix(timestampInt, 0)

				s.appendMetric(Metric{path, value, timestamp})
			}
		}(conn)
	}
}
func (s *MockServer) close() {
	if s.l != nil {
		s.l.Close()
		s.l = nil
	}
}

func (s *MockServer) HasErrors() bool {
	s.errorsMut.Lock()
	defer s.errorsMut.Unlock()

	return len(s.errors) > 0
}

func (s *MockServer) Errors() []error {
	s.errorsMut.Lock()
	defer s.errorsMut.Unlock()

	return s.errors
}

func (s *MockServer) appendError(errs ...error) {
	s.errorsMut.Lock()
	defer s.errorsMut.Unlock()

	s.errors = append(s.errors, errs...)
}

func (s *MockServer) Metrics() []Metric {
	s.metricsMut.Lock()
	defer s.metricsMut.Unlock()

	return s.metrics
}

func (s *MockServer) appendMetric(metrics ...Metric) {
	s.metricsMut.Lock()
	defer s.metricsMut.Unlock()

	s.metrics = append(s.metrics, metrics...)
}
