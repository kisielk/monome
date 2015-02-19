// Package monome implements an OSC interfaces to monome devices from monome.org
package monome

import (
	"errors"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/hypebeast/go-osc/osc"
)

var (
	// ConnectTimeout sets the timeout used by the Connect function.
	// It should not need to be changed in most cases.
	ConnectTimeout = 5 * time.Second

	// ErrTimeout is returned when the connection to a device cannot be established.
	ErrTimeout = errors.New("connection timed out")
)

// Connect is a utility method that establishes a connection to the first monome device it finds.
// The device sends key events to the given channel.
// It returns ErrTimeout if it can't connec to a device.
func Connect(prefix string, keyEvents chan KeyEvent) (*Device, error) {
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
		d, err := DialDevice(":"+strconv.Itoa(int(ev.Port)), prefix, keyEvents)
		if err != nil {
			return nil, err
		}
		// Wait for the Id to become populated.
		for i := 0; i < 100; i++ {
			id := d.Id()
			if id != "" {
				return d, nil
			}
			time.Sleep(10 * time.Millisecond)
		}
		return nil, ErrTimeout
	case <-time.After(ConnectTimeout):
		return nil, ErrTimeout
	}
}

// oscConnection is a bi-directional OSC connection
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

// HostPort returns the local OSC server host and port.
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

// Close terminates the OSC connection.
func (c *oscConnection) Close() error {
	return c.serverConn.Close()
}

// SerialOsc represents an OSC connection to the monome "serialosc" application.
type SerialOsc struct {
	*oscConnection
	events chan DeviceEvent
}

// A DeviceEvent is a Monome device connection or disconnection event.
type DeviceEvent struct {
	Id      string
	Type    string
	Port    int
	Removed bool // True if the device is being disconnected, otherwise false.
}

// DialSerialOsc creates a connection to a serialosc instance at the given address.
// If an empty address is given it defaults to localhost:12002
// Device add and remove events are sent to the given channel.
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

// List requests a list of all monome devices serialosc is aware of.
// The results are sent to the DeviceEvent channel the connetion was initialized with.
func (s *SerialOsc) List() error {
	host, port := s.HostPort()
	return s.send("/serialosc/list", host, int32(port))
}

func (s *SerialOsc) handleAdd(msg *osc.Message) {
	event, ok := s.handleDeviceEvent(msg)
	if !ok {
		return
	}
	s.events <- event
}

func (s *SerialOsc) handleRemove(msg *osc.Message) {
	event, ok := s.handleDeviceEvent(msg)
	if !ok {
		return
	}
	event.Removed = true
	s.events <- event
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
	port, ok := msg.Arguments[2].(int32)
	event.Port = int(port)
	return
}

// A KeyEvent is received for every key down or key up on a Monome device.
type KeyEvent struct {
	X     int
	Y     int
	State int // 1 for down, 0 for up.
}

// Device represents a connection to a Monome device.
type Device struct {
	*oscConnection
	mu       sync.RWMutex
	id       string
	width    int
	height   int
	prefix   string
	rotation int
	events   chan KeyEvent
}

// DialDevice connects to a Monome device using the given address.
// The address can be obtained from a SerialOsc.
// prefix is the OSC address prefix to be used by the local OSC server.
// If an empty prefix is given, it defaults to /gopher.
// KeyEvents which are received will be sent in to the given events channel.
func DialDevice(address, prefix string, events chan KeyEvent) (*Device, error) {
	conn, err := newOscConnection(address)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		prefix = "/gopher"
	}
	d := &Device{
		oscConnection: conn,
		prefix:        prefix,
		events:        events,
	}
	d.s.Handle(prefix+"/grid/key", d.handleKey)
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
	err = d.send("/sys/prefix", prefix)
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

// Height returns the height of the connected Monome device.
func (d *Device) Height() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.height
}

// Width returns the width of the connected Monome device.
func (d *Device) Width() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.width
}

// Id returns the id of the connected Monome device.
func (d *Device) Id() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.id
}

// Prefix returns the OSC prefinx being used in communication with the connected Monome device.
func (d *Device) Prefix() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.prefix
}

// Rotation returns the rotation of the connected Monome device.
func (d *Device) Rotation() int {
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
	d.width = int(width)
	d.height = int(height)
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
	d.rotation = int(rotation)
	d.mu.Unlock()
}

func (d *Device) handleKey(msg *osc.Message) {
	if msg.CountArguments() != 3 {
		return
	}
	x, ok := msg.Arguments[0].(int32)
	if !ok {
		return
	}
	y, ok := msg.Arguments[1].(int32)
	if !ok {
		return
	}
	state, ok := msg.Arguments[2].(int32)
	if !ok {
		return
	}
	d.events <- KeyEvent{int(x), int(y), int(state)}
}

func statesInterfaces(states []byte) []interface{} {
	in := make([]interface{}, len(states))
	for i := range states {
		in[i] = int32(states[i])
	}
	return in
}

func levelsInterfaces(levels []int) []interface{} {
	in := make([]interface{}, len(levels))
	for i := range levels {
		in[i] = int32(levels[i])
	}
	return in
}

// LEDSet sets the LED at (x, y) to the given state.
// State must be 1 for on or 0 for off.
func (d *Device) LEDSet(x, y, state int) error {
	return d.send(d.Prefix()+"/grid/led/set", int32(x), int32(y), int32(state))
}

// LEDAll sets all LEDs to the given state.
// State must be 1 for on or 0 for off.
func (d *Device) LEDAll(state int) error {
	return d.send(d.Prefix()+"/grid/led/all", int32(state))
}

// LEDMap sets an 8x8 grid of LEDs on the monome to the given states.
// The states are a bitmask with each byte representing one row and each bit representing the state of an LED in that row.
// xOffset and yOffset must be multiples of 8.
func (d *Device) LEDMap(xOffset, yOffset int, states [8]byte) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/map", int32(xOffset), int32(yOffset))
	m.Append(statesInterfaces(states[:])...)
	return d.sendMsg(m)
}

func (d *Device) LEDRow(xOffset, y int, states ...byte) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/row", int32(xOffset), int32(y))
	m.Append(statesInterfaces(states)...)
	return d.sendMsg(m)
}

func (d *Device) LEDCol(x, yOffset int, states ...byte) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/row", int32(x), int32(yOffset))
	m.Append(statesInterfaces(states)...)
	return d.sendMsg(m)
}

// LEDIntensity sets the intensity of the grid LEDs.
func (d *Device) LEDIntensity(i int) error {
	return d.send(d.Prefix()+"/grid/led/intensity", int32(i))
}

// LEDLevel sets the level of the LED at coordinates x, y. The value of level must be in the range [0, 15].
func (d *Device) LEDLevel(x, y, level int) error {
	return d.send(d.Prefix()+"/grid/led/level/set", int32(x), int32(y), int32(level))
}

// LEDLevelAll sets the level of all LEDs.
func (d *Device) LEDLevelAll(level int) error {
	return d.send(d.Prefix()+"/grid/led/level/all", int32(level))
}

// LEDLevelMap is like LEDMap but with control over the level.
func (d *Device) LEDLevelMap(xOffset, yOffset int, levels [64]int) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/level/map", int32(xOffset), int32(yOffset))
	m.Append(levelsInterfaces(levels[:])...)
	return d.sendMsg(m)
}

// LEDLevelRow is like LEDRow but with control over the level.
func (d *Device) LEDLevelRow(xOffset, y int, levels []int) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/level/row", int32(xOffset), int32(y))
	m.Append(levelsInterfaces(levels)...)
	return d.sendMsg(m)
}

// LEDLevelRow is like LEDCol but with control over the level.
func (d *Device) LEDLevelCol(x, yOffset int, levels []int) error {
	m := osc.NewMessage(d.Prefix()+"/grid/led/level/col", int32(x), int32(yOffset))
	m.Append(levelsInterfaces(levels)...)
	return d.sendMsg(m)
}
