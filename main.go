package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Config struct {
	Listen string
}

type NetbufOutput struct {
	env    *plugin.Env
	conf   *Config
	ln     net.Listener
	bufs   map[string][]string
	lock   sync.Mutex
	locks  map[string]sync.Mutex
	closed bool
}

func (p *NetbufOutput) Init(env *plugin.Env) (err error) {
	p.env = env
	p.conf = &Config{}
	p.bufs = make(map[string][]string)
	p.locks = make(map[string]sync.Mutex)
	return env.ReadConfig(p.conf)
}

func (p *NetbufOutput) Start() (err error) {
	if p.ln, err = net.Listen("tcp", p.conf.Listen); err != nil {
		return
	}
	go p.accepter()
	return
}

func (p *NetbufOutput) Encode(ev *message.Event) (buffer.Sizer, error) {
	t := ev.Time.Format("2006/01/02 15:04:05 MST")
	s := buffer.StringItem(fmt.Sprintf("%s %s %s", t, ev.Tag, ev.Record["message"]))
	return s, nil
}

func (p *NetbufOutput) Write(l []buffer.Sizer) (int, error) {
	for name, lock := range p.locks {
		lock.Lock()
		for _, s := range l {
			p.bufs[name] = append(p.bufs[name], string(s.(buffer.StringItem)))
		}
		lock.Unlock()
	}
	return len(l), nil
}

func (p *NetbufOutput) Close() error {
	p.closed = true
	return p.ln.Close()
}

func (p *NetbufOutput) accepter() {
	for !p.closed {
		conn, err := p.ln.Accept()
		if err != nil {
			continue
		}
		go p.handler(conn)
	}
}

func (p *NetbufOutput) handler(conn net.Conn) {
	defer conn.Close()
	remote, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	p.env.Log.Infof("Connected from %s", remote)
	lock, ok := p.locks[remote]
	if !ok {
		p.env.Log.Infof("No buffer found for %s, create now", remote)
		p.addBuffer(remote)
		return
	}

	lock.Lock()
	defer lock.Unlock()

	buf := p.bufs[remote]
	p.env.Log.Infof("Send %d line to %s", len(buf), remote)
	for i, s := range buf {
		if _, err := conn.Write([]byte(s + "\n")); err != nil {
			p.env.Log.Warning(err)
			p.bufs[remote] = buf[i:]
			return
		}
	}
	p.bufs[remote] = buf[:0]
}

func (p *NetbufOutput) addBuffer(name string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.bufs[name]; ok {
		return
	}
	p.bufs[name] = nil
	p.locks[name] = sync.Mutex{}
}

func main() {
	plugin.New("out-netbuf", func() plugin.Plugin {
		return &NetbufOutput{}
	}).Run()
}
