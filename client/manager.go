package client

import (
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/simpleiot/simpleiot/data"
)

// Manager manages a node type, watches for changes, adds/removes instances that get
// added/deleted
type Manager[T any] struct {
	// initial state
	nc        *nats.Conn
	root      string
	nodeType  string
	construct func(*nats.Conn, T) Client

	// synchronization fields
	stop       chan struct{}
	chScan     chan struct{}
	chAction   chan func()
	chDeleteCS chan string

	clientStates map[string]*clientState[T]

	// subscription to listen for new points
	upSub *nats.Subscription
}

// NewManager takes constructor for a node client and returns a Manager for that client
// The Node Type is inferred from the Go type passed in, so you must name Go client
// Types to manage the node type definitions.
func NewManager[T any](nc *nats.Conn, root string,
	construct func(nc *nats.Conn, config T) Client) *Manager[T] {
	var x T
	nodeType := reflect.TypeOf(x).Name()
	nodeType = strings.ToLower(nodeType[0:1]) + nodeType[1:]

	return &Manager[T]{
		nc:           nc,
		root:         root,
		nodeType:     nodeType,
		construct:    construct,
		stop:         make(chan struct{}),
		chScan:       make(chan struct{}),
		chAction:     make(chan func()),
		chDeleteCS:   make(chan string),
		clientStates: make(map[string]*clientState[T]),
	}
}

// Start node manager. This function looks for children of a certain node type.
// When new nodes are found, the data is decoded into the client type config, and the
// constructor for the node client is called. This call blocks until Stop is called.
func (m *Manager[T]) Start() error {
	// TODO: it may make sense at some point to have a special topic
	// for new nodes so that all client managers don't have to listen
	// to all points
	var err error
	m.upSub, err = m.nc.Subscribe("up.none.>", func(msg *nats.Msg) {
		points, err := data.PbDecodePoints(msg.Data)
		if err != nil {
			log.Println("Error decoding points")
			return
		}

		for _, p := range points {
			if p.Type == data.PointTypeNodeType {
				m.chScan <- struct{}{}
			}
		}
	})

	if err != nil {
		return err
	}

	err = m.scan()
	if err != nil {
		log.Println("Error scanning for new nodes: ", err)
	}

	shutdownTimer := time.NewTimer(time.Hour)
	shutdownTimer.Stop()

	stopping := false

	scan := func() {
		if stopping {
			return
		}

		err := m.scan()
		if err != nil {
			log.Println("Error scanning for new nodes: ", err)
		}
	}

done:
	for {
		select {
		case <-m.stop:
			stopping = true
			m.upSub.Unsubscribe()
			if len(m.clientStates) > 0 {
				for _, c := range m.clientStates {
					c.stop(err)
				}
				shutdownTimer.Reset(time.Second * 5)
			} else {
				break done
			}
		case f := <-m.chAction:
			f()
		case <-time.After(time.Minute):
			scan()
		case <-m.chScan:
			scan()
		case key := <-m.chDeleteCS:
			delete(m.clientStates, key)
			if stopping {
				if len(m.clientStates) <= 0 {
					break done
				}
			} else {
				// client may have exitted itself due to child
				// node changes so scan to re-initialize it again
				scan()
			}
		case <-shutdownTimer.C:
			// FIXME: should we return an error here?
			log.Println("BUG: Client manager: not all clients shutdown for node type: ", m.nodeType)
			for _, v := range m.clientStates {
				log.Println("Client stuck for node: ", v.node.ID)
			}
			break done
		}
	}

	return nil
}

// Stop manager. This also stops all registered clients and causes Start to exit.
func (m *Manager[T]) Stop(err error) {
	m.stop <- struct{}{}
}

func (m *Manager[T]) scan() error {
	children, err := GetNodeChildren(m.nc, m.root, m.nodeType, false, false)

	if err != nil {
		return err
	}

	if len(children) < 0 {
		return nil
	}

	found := make(map[string]bool)

	// create new nodes
	for _, n := range children {
		key := mapKey(n)
		found[key] = true

		if _, ok := m.clientStates[key]; ok {
			continue
		}

		cs := newClientState(m.nc, m.construct, n)

		m.clientStates[key] = cs

		go func() {
			err := cs.start()

			if err != nil {
				log.Printf("clientState error %v: %v\n", m.nodeType, err)
			}

			m.chDeleteCS <- key
		}()
	}

	// remove nodes that have been deleted
	for key, client := range m.clientStates {
		if _, ok := found[key]; ok {
			continue
		}

		// bus was deleted so close and clear it
		log.Println("removing client node: ", m.clientStates[key].node.ID)
		client.stop(nil)
	}

	return nil
}
