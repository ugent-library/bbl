package catbird

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxlisten"
)

var DefaultMessageBuffer = 16

type HubOpts struct {
	MessageBuffer int
}

type Hub struct {
	pool          *pgxpool.Pool
	shutdown      chan struct{}
	messageBuffer int
	subscribers   map[string][]*subscriber
	mu            sync.RWMutex
}

type subscriber struct {
	topics []string
	msgs   chan Msg
}

type Msg struct {
	Topic string `json:"topic,omitempty"`
	Event string `json:"event,omitempty"`
	Data  string `json:"data,omitempty"`
}

func NewHub(pool *pgxpool.Pool, opts HubOpts) *Hub {
	if opts.MessageBuffer == 0 {
		opts.MessageBuffer = DefaultMessageBuffer
	}
	return &Hub{
		pool:          pool,
		shutdown:      make(chan struct{}),
		messageBuffer: opts.MessageBuffer,
		subscribers:   make(map[string][]*subscriber),
	}
}

type ConnectOpts struct {
	Topics        []string
	MessageBuffer int
}

func (h *Hub) Start(ctx context.Context) error {
	listener := &pgxlisten.Listener{
		Connect: func(ctx context.Context) (*pgx.Conn, error) {
			conn, err := h.pool.Acquire(ctx)
			if err != nil {
				return nil, err
			}
			return conn.Conn(), nil
		},
	}
	listener.Handle("catbird", pgxlisten.HandlerFunc(func(ctx context.Context, not *pgconn.Notification, _ *pgx.Conn) error {
		var msg Msg
		if err := json.Unmarshal([]byte(not.Payload), &msg); err != nil {
			return err
		}
		h.send(msg)
		return nil
	}))
	return listener.Listen(ctx)
}

func (h *Hub) Shutdown() {
	close(h.shutdown) // TODO stop accepting new connections?
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
func (h *Hub) send(msg Msg) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if subs, ok := h.subscribers[msg.Topic]; ok {
		for _, sub := range subs {
			sub.msgs <- msg // TODO handle too slow (buffer full)
		}
	}
}

func (h *Hub) Send(ctx context.Context, msg Msg) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = h.pool.Exec(ctx, `select pg_notify($1, $2)`, "catbird", b)
	return err
}

type Renderer interface {
	Render(context.Context, io.Writer) error
}

func (h *Hub) Render(ctx context.Context, topic, event string, r Renderer) error {
	var b strings.Builder
	if err := r.Render(ctx, &b); err != nil {
		return err
	}
	h.Send(ctx, Msg{
		Topic: topic,
		Event: event,
		Data:  b.String(),
	})
	return nil
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
