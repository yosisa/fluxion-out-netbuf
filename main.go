package main

import (
	"fmt"
	"net"
	"sync"

	"github.com/yosisa/fluxion/buffer"
	"github.com/yosisa/fluxion/message"
	"github.com/yosisa/fluxion/plugin"
)

type Buffer struct {
	sync.Mutex
	Items []string
}

type Config struct {
	Listen string
}

type NetbufOutput struct {
	env    *plugin.Env
	conf   *Config
	ln     net.Listener
	bufs   map[string]*Buffer
	lock   sync.Mutex
	closed bool
}

func (p *NetbufOutput) Init(env *plugin.Env) (err error) {
	p.env = env
	p.conf = &Config{}
	p.bufs = make(map[string]*Buffer)
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
	for _, buf := range p.bufs {
		buf.Lock()
		for _, s := range l {
			buf.Items = append(buf.Items, string(s.(buffer.StringItem)))
		}
		buf.Unlock()
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
	buf, ok := p.bufs[remote]
	if !ok {
		p.env.Log.Infof("No buffer found for %s, create now", remote)
		p.addBuffer(remote)
		return
	}

	buf.Lock()
	defer buf.Unlock()

	p.env.Log.Infof("Send %d line to %s", len(buf.Items), remote)
	for i, s := range buf.Items {
		if _, err := conn.Write([]byte(s + "\n")); err != nil {
			p.env.Log.Warning(err)
			buf.Items = buf.Items[i:]
			return
		}
	}
	buf.Items = buf.Items[:0]
}

func (p *NetbufOutput) addBuffer(name string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.bufs[name]; ok {
		return
	}
	p.bufs[name] = &Buffer{}
}

func main() {
	plugin.New("out-netbuf", func() plugin.Plugin {
		return &NetbufOutput{}
	}).Run()
}
