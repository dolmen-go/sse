package sse_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/mubit/sse"
	"github.com/stretchr/testify/assert"
)

// Timeouts after the specified milliseconds
func timeout(ms time.Duration) <-chan bool {
	ch := make(chan bool, 1)
	go func() {
		time.Sleep(ms * time.Millisecond)
		ch <- true
	}()
	return ch
}

// Extracts decoder from a string
func newDecoder(data string) <-chan sse.Event {
	reader := bytes.NewReader([]byte(data))
	return sse.DefaultDecoder.Decode(reader)
}

// Attempts to consume an event from the decoding stream. Fails
// on timeouts or closed channel.
func consume(t *testing.T, decoder <-chan sse.Event) sse.Event {
	select {
	case ev, ok := <-decoder:
		if !ok {
			assert.Fail(t, "no more decoder to dispatch")
		}
		return ev
	case <-timeout(1000):
		assert.Fail(t, "timeout reached before dispatching event")
		return nil
	}
}

func TestEventNameAndData(t *testing.T) {
	decoder := newDecoder("event: some event\r\ndata: some event value\r\n\n")
	ev := consume(t, decoder)
	assert.Equal(t, "some event", ev.Name())
	assert.Equal(t, "some event value", string(ev.Data()))
}

func TestEventNameAndDataManyEvents(t *testing.T) {
	decoder := newDecoder("event: first event\r\ndata: first value\r\n\nevent: second event\r\ndata: second value\r\n\n")
	ev1 := consume(t, decoder)
	assert.Equal(t, "first event", ev1.Name())
	assert.Equal(t, "first value", string(ev1.Data()))
	ev2 := consume(t, decoder)
	assert.Equal(t, "second event", ev2.Name())
	assert.Equal(t, "second value", string(ev2.Data()))
}

func TestStocksExample(t *testing.T) {
	decoder := newDecoder("data: YHOO\ndata: +2\ndata: 10\n\n")
	ev := consume(t, decoder)
	assert.Equal(t, "YHOO\n+2\n10", string(ev.Data()))
}

func TestFirstWhitespaceIsIgnored(t *testing.T) {
	decoder := newDecoder("data: first\n\ndata: second\n\n")
	ev1 := consume(t, decoder)
	assert.Equal(t, "first", string(ev1.Data()))
	ev2 := consume(t, decoder)
	assert.Equal(t, "second", string(ev2.Data()))
}

func TestOnlyOneWhitespaceIsIgnored(t *testing.T) {
	decoder := newDecoder("data:   first\n\n") // 3 whitespaces
	ev := consume(t, decoder)
	assert.Equal(t, "  first", string(ev.Data())) // 2 whitespaces
}

func TestEventsWithNoDataThenWithNewLine(t *testing.T) {
	decoder := newDecoder("data\n\ndata\ndata\n\ndata:")
	ev1 := consume(t, decoder)
	assert.Equal(t, "", string(ev1.Data()))
	ev2 := consume(t, decoder)
	assert.Equal(t, "\n", string(ev2.Data()))
}

func TestCommentIsIgnoredAndDataIsNot(t *testing.T) {
	decoder := newDecoder(": test stream\n\ndata: first event\nid: 1\n\ndata:second event\nid\n\ndata:  third event\n\n")
	ev1 := consume(t, decoder)
	assert.Equal(t, "1", ev1.ID())
	assert.Equal(t, "first event", string(ev1.Data()))
	ev2 := consume(t, decoder)
	assert.Equal(t, "", ev2.ID())
	assert.Equal(t, "second event", string(ev2.Data()))
	ev3 := consume(t, decoder)
	assert.Equal(t, "", ev3.ID())
	assert.Equal(t, " third event", string(ev3.Data()))
}

func TestOneLineDataParseWithDoubleRN(t *testing.T) {
	decoder := newDecoder("data: this is a test\r\n\r\n")
	ev := consume(t, decoder)
	assert.Equal(t, "this is a test", string(ev.Data()))
}

func TestOneLineDataParseWithoutDoubleRN(t *testing.T) {
	decoder := newDecoder("data: this is a test\r\n\n")
	ev := consume(t, decoder)
	assert.Equal(t, "this is a test", string(ev.Data()))
}

func TestTwoLinesDataParseWithRNAndDoubleRN(t *testing.T) {
	decoder := newDecoder("data: this is \r\ndata: a test\r\n\r\n")
	ev := consume(t, decoder)
	assert.Equal(t, "this is \na test", string(ev.Data()))
}
