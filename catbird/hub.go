package catbird

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
)

var DefaultMessageBuffer = 16

type HubOpts struct {
	MessageBuffer int
}

type Hub struct {
	shutdown      chan struct{}
	messageBuffer int
	subscribers   map[string][]*subscriber
	mu            sync.RWMutex
}

type subscriber struct {
	topics []string
	msgs   chan Msg
}

func NewHub(opts HubOpts) *Hub {
	if opts.MessageBuffer == 0 {
		opts.MessageBuffer = DefaultMessageBuffer
	}
	return &Hub{
		shutdown:      make(chan struct{}),
		messageBuffer: opts.MessageBuffer,
		subscribers:   make(map[string][]*subscriber),
	}
}

type ConnectOpts struct {
	Topics        []string
	MessageBuffer int
}

func (h *Hub) Shutdown() {
	close(h.shutdown)
}

func (h *Hub) ConnectSSE(w http.ResponseWriter, r *http.Request, opts ConnectOpts) error {
	if opts.MessageBuffer == 0 {
		opts.MessageBuffer = h.messageBuffer
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return errors.New("could not upgrade sse connection")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()

	msgs := make(chan Msg, opts.MessageBuffer)

	sub := &subscriber{
		topics: opts.Topics,
		msgs:   msgs,
	}

	h.addSubscriber(sub)
	defer h.removeSubscriber(sub)

	for {
		select {
		case <-h.shutdown:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-msgs:
			if err := writeMsg(w, msg); err != nil {
				return err
			}
			flusher.Flush()
		}
	}
}

func (h *Hub) addSubscriber(sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, topic := range sub.topics {
		h.subscribers[topic] = append(h.subscribers[topic], sub)
	}
}

func (h *Hub) removeSubscriber(sub *subscriber) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, topic := range sub.topics {
		s := h.subscribers[topic]
		if i := slices.Index(s, sub); i >= 0 {
			h.subscribers[topic] = slices.Delete(s, i, i+1)
		}
	}

	close(sub.msgs)
}

type Msg struct {
	Event string
	Data  string
}

var newline = []byte{'\n'}
var eventField = []byte("event: ")
var dataField = []byte("data: ")

func writeMsg(w http.ResponseWriter, msg Msg) (err error) {
	b := &bytes.Buffer{} // TODO bufpool

	if msg.Event != "" {
		if _, err = b.Write(eventField); err != nil {
			return
		}
		if _, err = b.Write([]byte(msg.Event)); err != nil {
			return
		}
		if _, err = b.Write(newline); err != nil {
			return
		}
	}

	sc := bufio.NewScanner(strings.NewReader(msg.Data))
	for sc.Scan() {
		if _, err = b.Write(dataField); err != nil {
			return
		}
		if _, err = b.Write(sc.Bytes()); err != nil {
			return
		}
		if _, err = b.Write(newline); err != nil {
			return
		}
	}

	_, err = b.Write(newline)

	if err == nil {
		w.Write(b.Bytes())
	}

	return
}

func (h *Hub) Send(topic string, msg Msg) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if subs, ok := h.subscribers[topic]; ok {
		for _, sub := range subs {
			sub.msgs <- msg // TODO handle too slow (buffer full)
		}
	}
}

type Renderer interface {
	Render(context.Context, io.Writer) error
}

func (h *Hub) Render(ctx context.Context, topic, event string, r Renderer) error {
	var b strings.Builder
	if err := r.Render(ctx, &b); err != nil {
		return err
	}
	h.Send(topic, Msg{
		Event: event,
		Data:  b.String(),
	})
	return nil
}
