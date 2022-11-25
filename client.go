// Package graphigo provides a simple go client for the graphite monitoring tool.
// See http://graphite.readthedocs.org/en/latest/overview.html
package graphigo

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultDialTimeout is used when connecting to a graphite server.
	DefaultDialTimeout = 5 * time.Second

	// DefaultWriteTimeout is used when sending metrics.
	DefaultWriteTimeout = 5 * time.Second

	// DefaultGraphitePort is used when no port is specified in the address.
	DefaultGraphitePort = "2003"
)

type Config struct {
	// DialTimeout is used when connecting to the graphite server.
	DialTimeout time.Duration

	// WriteTimeout is used when sending metrics using Send().
	WriteTimeout time.Duration

	// Prefix is prepended to all metric names. If not present, a dot is automatically appended.
	Prefix string
}

func NewClient(address string, configFns ...func(c *Config)) (*Client, error) {
	config := Config{
		DialTimeout:  DefaultDialTimeout,
		WriteTimeout: DefaultWriteTimeout,
	}

	for _, fn := range configFns {
		fn(&config)
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil && strings.Contains(err.Error(), "missing port") {
		host = address
		port = DefaultGraphitePort
	} else if err != nil {
		return nil, fmt.Errorf("error parsing address: %v", err.Error())
	}

	prefix := config.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}

	return &Client{
		host: host, port: port,

		dialTimeout:  config.DialTimeout,
		writeTimeout: config.WriteTimeout,

		prefix: prefix,
	}, nil
}

// Client is a simple TCP client for the graphite monitoring tool.
type Client struct {
	host, port string

	mut sync.Mutex

	dialTimeout  time.Duration
	writeTimeout time.Duration
	prefix       string
	conn         net.Conn
}

// Send sends metrics to the graphite server. Send automatically establishes a connection if necessary.
func (c *Client) Send(metrics ...Metric) (err error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	defer func() {
		if err != nil {
			_ = c.close()
		}
	}()

	if err := c.connect(); err != nil {
		return err
	}

	var buf bytes.Buffer
	for i := range metrics {
		if metrics[i].Path == "" {
			return fmt.Errorf("no path supplied for metrics[%d]: %s", i, metrics[i])
		}
		if metrics[i].Timestamp.IsZero() {
			return fmt.Errorf("timestamp for metrics[%d] is zero: %s", i, metrics[i])
		}
		_, err := fmt.Fprintf(&buf, "%s%s %v %d\n", c.prefix, metrics[i].Path, metrics[i].Value, metrics[i].Timestamp.Unix())
		if err != nil {
			return fmt.Errorf("error writing metric to buffer: %w", err)
		}
	}

	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return fmt.Errorf("error setting write deadline: %w", err)
	}

	if _, err := c.conn.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("error sending metrics: %w", err)
	}

	return nil
}

func (c *Client) connect() error {
	if c.conn != nil {
		return nil
	}

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(c.host, c.port), c.dialTimeout)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

// Close closes the network connection used by Client. If no connection is established, Close is a no-op.
func (c *Client) Close() error {
	c.mut.Lock()
	defer c.mut.Unlock()

	return c.close()
}

func (c *Client) close() error {
	if c.conn == nil {
		return nil
	}
	err := c.conn.Close()
	c.conn = nil
	return err
}
