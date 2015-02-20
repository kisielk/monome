// Package monome implements an OSC interfaces to monome devices from monome.org
package monome

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/kisielk/go-osc/osc"
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
func Connect(prefix string, keyEvents chan KeyEvent) (*Grid, error) {
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
		g, err := DialGrid(":"+strconv.Itoa(int(ev.Port)), prefix, keyEvents)
		if err != nil {
			return nil, err
		}
		// Wait for the Id to become populated.
		for i := 0; i < 100; i++ {
			id := g.Id()
			if id != "" {
				return g, nil
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

// Grid represents a connection to a Monome device.
type Grid struct {
	*oscConnection
	mu       sync.RWMutex
	id       string
	width    int
	height   int
	prefix   string
	rotation int
	events   chan KeyEvent
}

// DialGrid connects to a Monome device using the given address.
// The address can be obtained from a SerialOsc.
// prefix is the OSC address prefix to be used by the local OSC server.
// If an empty prefix is given, it defaults to /gopher.
// KeyEvents which are received will be sent in to the given events channel.
func DialGrid(address, prefix string, events chan KeyEvent) (*Grid, error) {
	conn, err := newOscConnection(address)
	if err != nil {
		return nil, err
	}
	if prefix == "" {
		prefix = "/gopher"
	}
	d := &Grid{
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
func (g *Grid) Height() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.height
}

// Width returns the width of the connected Monome device.
func (g *Grid) Width() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.width
}

// Id returns the id of the connected Monome device.
func (g *Grid) Id() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.id
}

// Prefix returns the OSC prefinx being used in communication with the connected Monome device.
func (g *Grid) Prefix() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.prefix
}

// Rotation returns the rotation of the connected Monome device.
func (g *Grid) Rotation() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.rotation
}

func (g *Grid) handlePort(msg *osc.Message) {
	return
}

func (g *Grid) handleId(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	id, ok := msg.Arguments[0].(string)
	if !ok {
		return
	}
	g.mu.Lock()
	g.id = id
	g.mu.Unlock()
}

func (g *Grid) handleSize(msg *osc.Message) {
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
	g.mu.Lock()
	g.width = int(width)
	g.height = int(height)
	g.mu.Unlock()
}

func (g *Grid) handlePrefix(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	prefix, ok := msg.Arguments[0].(string)
	if !ok {
		return
	}
	g.mu.Lock()
	g.prefix = prefix
	g.mu.Unlock()
}

func (g *Grid) handleRotation(msg *osc.Message) {
	if msg.CountArguments() != 1 {
		return
	}
	rotation, ok := msg.Arguments[0].(int32)
	if !ok {
		return
	}
	g.mu.Lock()
	g.rotation = int(rotation)
	g.mu.Unlock()
}

func (g *Grid) handleKey(msg *osc.Message) {
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
	g.events <- KeyEvent{int(x), int(y), int(state)}
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
func (g *Grid) LEDSet(x, y, state int) error {
	return g.send(g.Prefix()+"/grid/led/set", int32(x), int32(y), int32(state))
}

// LEDAll sets all LEDs to the given state.
// State must be 1 for on or 0 for off.
func (g *Grid) LEDAll(state int) error {
	return g.send(g.Prefix()+"/grid/led/all", int32(state))
}

// LEDMap sets an 8x8 grid of LEDs on the monome to the given states.
// The states are a bitmask with each byte representing one row and each bit representing the state of an LED in that row.
// xOffset and yOffset must be multiples of 8.
func (g *Grid) LEDMap(xOffset, yOffset int, states [8]byte) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/map", int32(xOffset), int32(yOffset))
	m.Append(statesInterfaces(states[:])...)
	return g.sendMsg(m)
}

func (g *Grid) LEDRow(xOffset, y int, states ...byte) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/row", int32(xOffset), int32(y))
	m.Append(statesInterfaces(states)...)
	return g.sendMsg(m)
}

func (g *Grid) LEDCol(x, yOffset int, states ...byte) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/row", int32(x), int32(yOffset))
	m.Append(statesInterfaces(states)...)
	return g.sendMsg(m)
}

// LEDIntensity sets the intensity of the grid LEDs.
func (g *Grid) LEDIntensity(i int) error {
	return g.send(g.Prefix()+"/grid/led/intensity", int32(i))
}

// LEDLevel sets the level of the LED at coordinates x, y. The value of level must be in the range [0, 15].
func (g *Grid) LEDLevelSet(x, y, level int) error {
	return g.send(g.Prefix()+"/grid/led/level/set", int32(x), int32(y), int32(level))
}

// LEDLevelAll sets the level of all LEDs.
func (g *Grid) LEDLevelAll(level int) error {
	return g.send(g.Prefix()+"/grid/led/level/all", int32(level))
}

// LEDLevelMap is like LEDMap but with control over the level.
func (g *Grid) LEDLevelMap(xOffset, yOffset int, levels [64]int) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/level/map", int32(xOffset), int32(yOffset))
	m.Append(levelsInterfaces(levels[:])...)
	return g.sendMsg(m)
}

// LEDLevelRow is like LEDRow but with control over the level.
func (g *Grid) LEDLevelRow(xOffset, y int, levels []int) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/level/row", int32(xOffset), int32(y))
	m.Append(levelsInterfaces(levels)...)
	return g.sendMsg(m)
}

// LEDLevelRow is like LEDCol but with control over the level.
func (g *Grid) LEDLevelCol(x, yOffset int, levels []int) error {
	m := osc.NewMessage(g.Prefix()+"/grid/led/level/col", int32(x), int32(yOffset))
	m.Append(levelsInterfaces(levels)...)
	return g.sendMsg(m)
}

// LEDBuffer can be used to buffer LED changes to a grid.
// It supports all the same LED operations as a Grid, but doesn't send anything
// until Render is called.
type LEDBuffer struct {
	buf    []int
	width  int
	height int
}

func NewLEDBuffer(width, height int) *LEDBuffer {
	return &LEDBuffer{
		buf:    make([]int, width*height),
		width:  width,
		height: height,
	}
}

func (b *LEDBuffer) LEDSet(x, y, state int) error {
	b.LEDLevelSet(x, y, state*15)
	return nil
}

func (b *LEDBuffer) LEDAll(state int) error {
	b.LEDLevelAll(state * 15)
	return nil
}

func (b *LEDBuffer) LEDMap(xOffset, yOffset int, states [8]byte) error {
	for row, data := range states {
		b.LEDRow(xOffset, yOffset+row, data)
	}
	return nil
}

func (b *LEDBuffer) LEDRow(xOffset, y int, states ...byte) error {
	if len(states) > b.width-xOffset {
		panic(fmt.Errorf("too many states: %d. width (%d) - xOffset (%d) = %d",
			len(states), b.width, xOffset, b.width-xOffset))
	}

	for row, data := range states {
		d := uint(data)
		for x := 0; x < 8; x++ {
			state := d & 1 << uint(x)
			b.LEDSet(xOffset+x, y+row, int(state))
		}
	}
	return nil
}

func (b *LEDBuffer) LEDCol(x, yOffset int, states ...byte) error {
	if len(states) > b.height-yOffset {
		panic(fmt.Errorf("too many levels: %d. height (%d) - yOffset (%d) = %d",
			len(states), b.height, yOffset, b.height-yOffset))
	}

	for col, data := range states {
		d := uint(data)
		for y := 0; y < 8; y++ {
			state := d & 1 << uint(y)
			b.LEDSet(x+col, yOffset+y, int(state))
		}
	}
	return nil
}

func (b *LEDBuffer) LEDLevelSet(x, y, level int) error {
	b.buf[x+(y*b.width)] = level
	return nil
}

func (b *LEDBuffer) LEDLevelAll(level int) error {
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.height; x++ {
			b.buf[x+(y*b.width)] = level
		}
	}
	return nil
}

func (b *LEDBuffer) LEDLevelMap(xOffset, yOffset int, levels [64]int) error {
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			b.buf[x+xOffset+(y+yOffset)*b.width] = levels[x+y*8]
		}
	}
	return nil
}

func (b *LEDBuffer) LEDLevelRow(xOffset, y int, levels []int) error {
	if len(levels) > b.width-xOffset {
		panic(fmt.Errorf("too many levels: %d. width (%d) - xOffset (%d) = %d",
			len(levels), b.width, xOffset, b.width-xOffset))
	}

	for x, level := range levels {
		b.buf[xOffset+x+y*b.width] = level
	}
	return nil
}

func (b *LEDBuffer) LEDLevelCol(x, yOffset int, levels []int) error {
	if len(levels) > b.height-yOffset {
		panic(fmt.Errorf("too many levels: %d. height (%d) - yOffset (%d) = %d",
			len(levels), b.height, yOffset, b.height-yOffset))
	}

	for y, level := range levels {
		b.buf[x+(y+yOffset)*b.width] = level
	}
	return nil
}

func (b *LEDBuffer) Render(g *Grid) error {
	for yOff := 0; yOff < b.height; yOff += 8 {
		for xOff := 0; xOff < b.width; xOff += 8 {
			err := g.LEDLevelMap(xOff, yOff, b.levelMap(xOff, yOff))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *LEDBuffer) levelMap(xOffset, yOffset int) [64]int {
	var m [64]int
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			m[x+y*8] = b.buf[x+xOffset+(y+yOffset)*b.width]
		}
	}
	return m
}
