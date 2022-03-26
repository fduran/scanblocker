package main

import (
	"errors"
	"github.com/google/gopacket/layers"
)

// connection attempt per IP: timestamp and host port
type conn struct {
	timestamp int64          // epoch, 8  bytes
	port      layers.TCPPort // uint16 , 2 bytes
}

type Queue struct {
	data    []conn
	maxSize uint16 // worst (biggest) case: array with all 65535 ports
}

// returns a new instance of a Queue
func NewQueue(size uint16) (*Queue, error) {
	if size == 0 || size > 65535 {
		return nil, errors.New("Queue size must be 1 - 65535")
	}

	return &Queue{
		data:    make([]conn, 0, size), // slice: initial capacity = size
		maxSize: size,
	}, nil
}

// add a new element and if slice is full, drop oldest element
func (q *Queue) Add(n conn) *Queue {
	// Queue len can grow up to its maxSize
	if uint16(len(q.data)) < q.maxSize {
		q.data = append(q.data, n) // add to slice up to capacity
	} else {
		q.data = q.data[1:]        // drop first item data[0]
		q.data = append(q.data, n) // append new element
	}

	return q
}
