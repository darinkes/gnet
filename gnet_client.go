package gnet

import (
	"log"
	"net"
	"sync"

	"github.com/panjf2000/gnet/netpoll"
	"github.com/panjf2000/gnet/ringbuffer"
)

type Client struct {
	connection       *connection
	eventHandler     EventHandler
	opts             *Options
	bytesPool        sync.Pool
	mainLoop         *loop
	wg               sync.WaitGroup
	cond             *sync.Cond
	once             sync.Once
	conn             *conn
	codec            ICodec
}

func (cl *Client) Write(b []byte) {
	cl.conn.write(b)
}

func (cl *Client) LocalAddr() net.Addr {
	return cl.connection.pconn.LocalAddr()
}

func (cl *Client) RemoteAddr() net.Addr {
	return cl.connection.pconn.RemoteAddr()
}

func (cl *Client) Close() {
	cl.stop()
}

func (cl *Client) waitForShutdown() {
	cl.cond.L.Lock()
	cl.cond.Wait()
	cl.cond.L.Unlock()
}

func (cl *Client) signalShutdown() {
	cl.once.Do(func() {
		cl.cond.L.Lock()
		cl.cond.Signal()
		cl.cond.L.Unlock()
	})
}

func (cl *Client) activateReactors() error {
	if p, err := netpoll.OpenPoller(); err == nil {
		lp := &loop{
			idx:         0,
			poller:      p,
			packet:      make([]byte, 0xFFFF),
			connections: make(map[int]*conn),
			cl:          cl,
		}
		_ = lp.poller.AddRead(cl.connection.fd)

		cl.mainLoop = lp

		cl.conn = newClientConn(cl.connection.fd, cl.mainLoop)
		lp.connections[cl.connection.fd] = cl.conn

		// Start main reactor.
		cl.wg.Add(1)
		go func() {
			cl.activateMainReactor()
			cl.wg.Done()
		}()
	} else {
		return err
	}

	return nil
}

func (cl *Client) start() error {
	return cl.activateReactors()
}

func (cl *Client) stop() {
	// Wait on a signal for shutdown
	cl.waitForShutdown()

	if cl.mainLoop != nil {
		sniffError(cl.mainLoop.poller.Trigger(func() error {
			return errShutdown
		}))
	}

	// Wait on all loops to complete reading events
	cl.wg.Wait()

	if cl.mainLoop != nil {
		sniffError(cl.mainLoop.poller.Close())
	}
}

func connect(eventHandler EventHandler, connection *connection, options *Options) error {
	cl := new(Client)
	cl.eventHandler = eventHandler
	cl.opts = options
	cl.connection = connection
	cl.cond = sync.NewCond(&sync.Mutex{})

	cl.bytesPool.New = func() interface{} {
		return ringbuffer.New(socketRingBufferSize)
	}

	cl.codec = func() ICodec {
		if options.Codec == nil {
			return new(BuiltInFrameCodec)
		}
		return options.Codec
	}()

	if err := cl.start(); err != nil {
		log.Printf("gnet client is stopping with error: %v\n", err)
		return err
	}
	defer cl.stop()

	switch cl.eventHandler.OnConnectionEstablished(cl) {
	case None:
	case Shutdown:
		return nil
	}

	return nil
}
