package udp

import (
	"errors"
	"net"
	"os"
	"sync"
	"time"

	"go.dedis.ch/cs438/transport"
)

const buffSize = 65000
const network = "udp"

// NewUDP returns a new udp transport implementation
func NewUDP() transport.Transport {
	return &UDP{}
}

// UDP implements a transport layer using UDP
type UDP struct {
}

// create the socket
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {

	// return the address of UDP end point
	UDPAddr, err := computeUDPAddr(address)
	if err != nil {
		return nil, err
	}

	// create UDP connection
	conn, err := createConnection(UDPAddr)
	if err != nil {
		return nil, err
	}

	return &Socket{
		conn: conn,

		ins:  packets{},
		outs: packets{},
	}, nil
}

// Socket implements a network socket using UDP
type Socket struct {
	sync.RWMutex
	conn *net.UDPConn

	ins  packets
	outs packets
}

// closes the connection
func (s *Socket) Close() error {
	return s.conn.Close()
}

// sends a msg to the destination
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	s.RLock()
	defer s.RUnlock()

	// set timeout
	err := s.setSendTimeout(timeout)
	if err != nil {
		return err
	}

	// return the address of UDP end point
	UDPAddr, err := computeUDPAddr(dest)
	if err != nil {
		return err
	}

	// transform a packet to something that can be sent over the network
	msg, err := pkt.Marshal()
	if err != nil {
		return err
	}

	// send message to UDP
	err = s.sendUDPPacket(msg, UDPAddr, timeout)
	if err != nil {
		return err
	}

	// add packet to sent packets
	s.outs.add(pkt)

	return nil
}

// blocks until a packet is received, or the timeout is reached.
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	s.RLock()
	defer s.RUnlock()

	buff := make([]byte, buffSize)
	pkt := transport.Packet{}

	// set timeout
	err := s.setRecvTimeout(timeout)
	if err != nil {
		return pkt, err
	}

	// read packet from UDP
	nb, err := s.receiveUDPPacket(buff, timeout)
	if err != nil {
		return pkt, err
	}

	// transform a marshaled packet to an actual packet
	err = (&pkt).Unmarshal(buff[:nb])
	if err != nil {
		return pkt, err
	}

	// add packet to received packets
	s.ins.add(pkt)

	return pkt, nil
}

// returns the address assigned
func (s *Socket) GetAddress() string {
	return s.conn.LocalAddr().String()
}

// returns all the messages received so far
func (s *Socket) GetIns() []transport.Packet {
	return s.ins.getAll()
}

// returns all the messages sent so far
func (s *Socket) GetOuts() []transport.Packet {
	return s.outs.getAll()
}

// returns the address of UDP end point
func computeUDPAddr(addr string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr(network, addr)
}

// creates UDP connection
func createConnection(addr *net.UDPAddr) (*net.UDPConn, error) {
	return net.ListenUDP(network, addr)
}

// sets timeout for sending packets
func (s *Socket) setSendTimeout(timeout time.Duration) error {
	var err error
	if timeout == 0 {
		err = s.conn.SetWriteDeadline(time.Time{})
	} else {
		err = s.conn.SetWriteDeadline(time.Now().Add(timeout))
	}
	return err
}

// sets timeout for receiving packets
func (s *Socket) setRecvTimeout(timeout time.Duration) error {
	var err error
	if timeout == 0 {
		err = s.conn.SetReadDeadline(time.Time{})
	} else {
		err = s.conn.SetReadDeadline(time.Now().Add(timeout))
	}
	return err
}

// sends message to UDP and manage timeout error
func (s *Socket) sendUDPPacket(msg []byte, UDPAddr *net.UDPAddr, timeout time.Duration) error {
	_, err := s.conn.WriteToUDP(msg, UDPAddr)
	if err != nil && errors.Is(err, os.ErrDeadlineExceeded) {
		err = transport.TimeoutError(timeout)
	}
	return err
}

// receives message from UDP and manage timeout error
func (s *Socket) receiveUDPPacket(buff []byte, timeout time.Duration) (int, error) {
	nb, _, err := s.conn.ReadFromUDP(buff)
	if err != nil && errors.Is(err, os.ErrDeadlineExceeded) {
		err = transport.TimeoutError(timeout)
	}
	return nb, err
}

type packets struct {
	sync.Mutex
	data []transport.Packet
}

func (p *packets) add(pkt transport.Packet) {
	p.Lock()
	defer p.Unlock()

	p.data = append(p.data, pkt.Copy())
}

func (p *packets) getAll() []transport.Packet {
	p.Lock()
	defer p.Unlock()

	res := make([]transport.Packet, len(p.data))

	for i, pkt := range p.data {
		res[i] = pkt.Copy()
	}

	return res
}
