package main

import (
	"testing"
	"time"

	"github.com/google/gopacket/layers"
)

func TestLeakyQueue(t *testing.T) {
	// test queue size out of bounds edge case
	// valid uint16 but bad queue size 0
	_, err := NewQueue(0)
	if err == nil {
		t.Errorf("NewQueue(0) not trigering error")
	}

	// we could increment this
	tUnix := time.Now().Unix()

	// default size (number of different ports to keep track of) is 3
	// let's check say 1 - 10
	var size uint16 = 0
	var port uint16 = 0
	for size = 1; size < 10; size++ {
		q, _ := NewQueue(size)

		// let's create conns up to the number of slots
		for port = 1; port <= size; port++ {
			q.Add(conn{tUnix, layers.TCPPort(port)})

			// length of data should after every addition be the number of additions
			if uint16(len(q.data)) != port {
				t.Errorf("For size %v got queue length %d, want %d", size, len(q.data), port)
			}
		}

		// let's create more conns than slots
		for port = size; port < 100; port++ {
			q.Add(conn{tUnix, layers.TCPPort(port)})

			// length of data should always be the (max) size
			if uint16(len(q.data)) != size {
				t.Errorf("Got queue length %d, want %d", len(q.data), size)
			}
		}

		// contents should be the last (size) conns when conns > size
		// going up to < 100 on size 3 for ex it should be the last 3: 97, 98, 99
		if size == 3 {
			want := []conn{
				{tUnix, 97},
				{tUnix, 98},
				{tUnix, 99},
			}
			for i := 1; i < int(size); i++ {
				if q.data[i] != want[i] {
					t.Errorf("Queue data error: got %d, want %d", q.data[i], want[i])
				}
			}
		}
	}
}
