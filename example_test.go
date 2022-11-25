package graphigo_test

import (
	"time"

	"github.com/shasderias/graphigo"
)

func Example() {
	// The port number can be omitted if connecting to the default port (2003).
	client, err := graphigo.NewClient("localhost:2003", func(c *graphigo.Config) {
		// These are the default values. If you do not want to change them, the second argument
		// can be omitted, i.e. graphigo.NewClient("localhost") is sufficient.

		c.DialTimeout = 5 * time.Second

		c.WriteTimeout = 5 * time.Second

		// If Prefix is not empty, it will be prepended to all metrics sent.
		// A dot separator automatically added if required.
		c.Prefix = ""
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	// Send automatically establishes a connection if necessary. There is no Connect().
	// All fields in Metric are required, Timestamp cannot be zero.
	if err := client.Send(graphigo.Metric{Path: "path", Value: 3.14, Timestamp: time.Now()}); err != nil {
		panic(err)
	}

	// Send is variadic.
	if err := client.Send(
		graphigo.Metric{"over", 3.14, time.Now()},
		graphigo.Metric{"the", 137.035, time.Now()},
		graphigo.Metric{"hills", 6.626, time.Now()},
	); err != nil {
		panic(err)
	}
	if err := client.Send(
		[]graphigo.Metric{
			{"and", 299792458, time.Now()},
			{"far", 8.854, time.Now()},
			{"away", 1.602, time.Now()},
		}...,
	); err != nil {
		panic(err)
	}
}
