package catbird

import (
	"bufio"
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
		messageBuffer: opts.MessageBuffer,
		subscribers:   make(map[string][]*subscriber),
	}
}

type ConnectOpts struct {
	Topics        []string
	MessageBuffer int
}

func (h *Hub) ConnectSSE(w http.ResponseWriter, r *http.Request, opts ConnectOpts) error {
	if opts.MessageBuffer == 0 {
		opts.MessageBuffer = h.messageBuffer
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	disconnected := r.Context().Done()

	msgs := make(chan Msg, opts.MessageBuffer)

	sub := &subscriber{
		topics: opts.Topics,
		msgs:   msgs,
	}

	h.addSubscriber(sub)
	defer h.removeSubscriber(sub)

	res := http.NewResponseController(w)

	for {
		select {
		case <-disconnected:
			return nil
		case msg := <-msgs:
			if err := writeMsg(w, msg); err != nil {
				return err
			}
			if err := res.Flush(); err != nil {
				return err
			}
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
	if msg.Event != "" {
		if _, err = w.Write(eventField); err != nil {
			return
		}
		if _, err = w.Write(eventField); err != nil {
			return
		}
	}

	sc := bufio.NewScanner(strings.NewReader(msg.Data))
	for sc.Scan() {
		if _, err = w.Write(dataField); err != nil {
			return
		}
		if _, err = w.Write(sc.Bytes()); err != nil {
			return
		}
		if _, err = w.Write(newline); err != nil {
			return
		}
	}

	_, err = w.Write(newline)
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
