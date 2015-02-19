package monome

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var ErrTimeout = errors.New("connection timed out")

// Connect is a utility method that establishes a connection to the first monome device it finds.
// The device sends key events to the given channel.
func Connect(keyEvents chan KeyEvent) (*Device, error) {
	deviceEvents := make(chan DeviceEvent)
	so, err := DialSerialOsc("", deviceEvents)
	if err != nil {
		return nil, err
	}
	defer so.Close()
	err = so.List()
	if err != nil {
		return nil, err
	}
	select {
	case ev := <-deviceEvents:
		return DialDevice(":"+strconv.Itoa(int(ev.Port)), keyEvents)
	case <-time.After(5 * time.Second):
		return nil, ErrTimeout
	}
}

type oscConnection struct {
	c          *osc.Client
	s          *osc.Server
	serverConn net.PacketConn
}

func newOscConnection(address string) (*oscConnection, error) {
	c, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &osc.Server{}
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		return nil, err
	}
	go s.Serve(c)
	return &oscConnection{
		c:          osc.NewClient(host, p),
		serverConn: c,
		s:          s,
	}, nil
}

func (c *oscConnection) HostPort() (string, int) {
	host, p, _ := net.SplitHostPort(c.serverConn.LocalAddr().String())
	port, _ := strconv.Atoi(p)
	return host, port
}

func (c *oscConnection) send(address string, args ...interface{}) error {
	m := osc.NewMessage(address, args...)
	return c.sendMsg(m)
}

func (c *oscConnection) sendMsg(m *osc.Message) error {
	return c.c.Send(m)
}

func (c *oscConnection) Close() error {
	return c.serverConn.Close()
}

type SerialOsc struct {
	*oscConnection
	Events chan DeviceEvent
}

type DeviceEvent struct {
	Id      string
	Type    string
	Port    int32
	Removed bool
}

func DialSerialOsc(address string, events chan DeviceEvent) (*SerialOsc, error) {
	if address == "" {
		address = "localhost:12002"
	}
	conn, err := newOscConnection(address)
	s := &SerialOsc{conn, events}
	s.s.Handle("/serialosc/device", s.handleAdd)
	s.s.Handle("/serialosc/add", s.handleAdd)
	s.s.Handle("/serialosc/remove", s.handleRemove)
	return s, err
}

func (s *SerialOsc) List() error {
	host, port := s.HostPort()
	return s.send("/serialosc/list", host, int32(port))
}

func (s *SerialOsc) handleAdd(msg *osc.Message) {
	event, ok := s.handleDeviceEvent(msg)
	if !ok {
		return
	}
	s.Events <- event
}

func (s *SerialOsc) handleRemove(msg *osc.Message) {
	event, ok := s.handleDeviceEvent(msg)
	if !ok {
		return
	}
	event.Removed = true
	s.Events <- event
}

func (s *SerialOsc) handleDeviceEvent(msg *osc.Message) (event DeviceEvent, ok bool) {
	if msg.CountArguments() != 3 {
		return
	}
	event.Id, ok = msg.Arguments[0].(string)
	if !ok {
		return
	}
	event.Type, ok = msg.Arguments[1].(string)
	if !ok {
		return
	}
	event.Port, ok = msg.Arguments[2].(int32)
	return
}

type KeyHandler interface {
	HandleKey(x, y, s int)
}

type KeyEvent struct {
	X     int32
	Y     int32
	State int32
}

type Device struct {
	*oscConnection
	mu       sync.RWMutex
	id       string
	width    int32
	height   int32
	prefix   string
	rotation int32
	Events   chan KeyEvent
}

func DialDevice(address string, events chan KeyEvent) (*Device, error) {
	conn, err := newOscConnection(address)
	if err != nil {
		return nil, err
	}
	d := &Device{oscConnection: conn, Events: events}
	d.s.Handle("/manager/grid/key", d.handleKey)
	d.s.Handle("/sys/port", d.handlePort)
	d.s.Handle("/sys/id", d.handleId)
	d.s.Handle("/sys/size", d.handleSize)
	d.s.Handle("/sys/prefix", d.handlePrefix)
	d.s.Handle("/sys/rotation", d.handleRotation)
	host, port := d.HostPort()
	err = d.send("/sys/host", host)
	if err != nil {
		d.Close()
		return nil, err
	}
	err = d.send("/sys/port", int32(port))
	if err != nil {
		d.Close()
		return nil, err
	}
	err = d.send("/sys/info")
	if err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

func (d *Device) Height() int32 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.height
}

func (d *Device) Width() int32 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.width
}

func (d *Device) Id() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.id
}

func (d *Device) Prefix() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.prefix
}

func (d *Device) Rotation() int32 {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.rotation
}

func (d *Device) handlePort(msg *osc.Message) {
	return
}

func (d *Device) handleId(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	id, ok := msg.Arguments[0].(string)
	if !ok {
		return
	}
	d.mu.Lock()
	d.id = id
	d.mu.Unlock()
}

func (d *Device) handleSize(msg *osc.Message) {
	if msg.CountArguments() != 2 {
		return
	}
	width, ok := msg.Arguments[0].(int32)
	if !ok {
		return
	}
	height, ok := msg.Arguments[1].(int32)
	if !ok {
		return
	}
	d.mu.Lock()
	d.width = width
	d.height = height
	d.mu.Unlock()
}

func (d *Device) handlePrefix(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	prefix, ok := msg.Arguments[0].(string)
	if !ok {
		return
	}
	d.mu.Lock()
	d.prefix = prefix
	d.mu.Unlock()
}

func (d *Device) handleRotation(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	rotation, ok := msg.Arguments[0].(int32)
	if !ok {
		return
	}
	d.mu.Lock()
	d.rotation = rotation
	d.mu.Unlock()
}

func (d *Device) handleKey(msg *osc.Message) {
	if msg.CountArguments() != 3 {
		return
	}
	var (
		event KeyEvent
		ok    bool
	)
	event.X, ok = msg.Arguments[0].(int32)
	if !ok {
		return
	}
	event.Y, ok = msg.Arguments[1].(int32)
	if !ok {
		return
	}
	event.State = msg.Arguments[2].(int32)
	if !ok {
		return
	}
	d.Events <- event
}

func statesInterfaces(states []uint8) []interface{} {
	in := make([]interface{}, len(states))
	for i := range states {
		in[i] = states[i]
	}
	return in
}

func levelsInterfaces(levels []int) []interface{} {
	in := make([]interface{}, len(levels))
	for i := range levels {
		in[i] = levels[i]
	}
	return in
}

func (d *Device) Set(x, y, state int32) error {
	return d.send("/manager/grid/led/set", x, y, state)
}

func (d *Device) All(state int) error {
	return d.send("/grid/led/all", state)
}

func (d *Device) Map(xOffset, yOffset int, states [8]uint8) error {
	m := osc.NewMessage("/grid/led/map", xOffset, yOffset)
	m.Append(statesInterfaces(states[:])...)
	return d.sendMsg(m)
}

func (d *Device) Rows(xOffset, y int, states ...uint8) error {
	m := osc.NewMessage("/grid/led/row", xOffset, y)
	m.Append(statesInterfaces(states)...)
	return d.sendMsg(m)
}

func (d *Device) Cols(x, yOffset int, states ...uint8) error {
	m := osc.NewMessage("/grid/led/row", x, yOffset)
	m.Append(statesInterfaces(states)...)
	return d.sendMsg(m)
}

func (d *Device) Intensity(i int) error {
	return d.send("/grid/led/intensity", i)
}

func (d *Device) Level(x, y int, level int) error {
	return d.send("/grid/led/level/set", x, y, level)
}

func (d *Device) LevelAll(level int) error {
	return d.send("/grid/led/level/all", level)
}

func (d *Device) LevelMap(xOffset, yOffset int, levels [64]int) error {
	m := osc.NewMessage("/grid/led/level/map", xOffset, yOffset)
	m.Append(levelsInterfaces(levels[:])...)
	return d.sendMsg(m)
}

func (d *Device) LevelRows(xOffset, y int, levels []int) error {
	m := osc.NewMessage("/grid/led/level/row", xOffset, y)
	m.Append(levelsInterfaces(levels)...)
	return d.sendMsg(m)
}

func (d *Device) LevelCols(x, yOffset int, levels []int) error {
	m := osc.NewMessage("/grid/led/level/col", x, yOffset)
	m.Append(levelsInterfaces(levels)...)
	return d.sendMsg(m)
}
