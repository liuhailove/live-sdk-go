package jitter

import (
	"github.com/go-logr/logr"
	"github.com/livekit/protocol/logger"
	"github.com/pion/rtp"
	"sync"
	"time"
)

type Buffer struct {
	depacketizer    rtp.Depacketizer
	maxLate         uint32
	clockRate       uint32
	onPacketDropped func()
	logger          logger.Logger

	mu          sync.Mutex
	pool        *packet
	size        int
	initialized bool
	prevSN      uint16
	head        *packet
	tail        *packet

	maxSampleSize uint32
	minTS         uint32
}

type packet struct {
	prev, next        *packet
	start, end, reset bool
	packet            *rtp.Packet
}

func NewBuffer(depacketizer rtp.Depacketizer, clockRate uint32, maxLatency time.Duration, opts ...Option) *Buffer {
	b := &Buffer{
		depacketizer: depacketizer,
		maxLate:      uint32(float64(maxLatency) / float64(time.Second) * float64(clockRate)),
		clockRate:    clockRate,
		logger:       logger.LogRLogger(logr.Discard()),
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *Buffer) UpdateMaxLatency(maxLatency time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	maxLate := uint32(float64(maxLatency) / float64(time.Second) * float64(b.clockRate))
	b.minTS += b.maxLate - maxLate
	b.maxLate = maxLate
}

func (b *Buffer) Push(pkt *rtp.Packet) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var start, end bool
	if len(pkt.Payload) == 0 {
		start = true
		end = true
	} else {
		start = b.depacketizer.IsPartitionHead(pkt.Payload)
		end = b.depacketizer.IsPartitionTail(pkt.Marker, pkt.Payload)
	}

	p := b.newPacket(start, end, pkt)

	if b.tail == nil {
		// list is empty
		if !b.initialized {
			b.initialized = true
			p.reset = p.start
		} else {
			p.reset = p.start && outsideRange(b.prevSN, pkt.SequenceNumber)
		}

		b.minTS = pkt.Timestamp - b.maxLate
		b.head = p
		b.tail = p
		return
	}

	d := b.tail.packet.SequenceNumber - pkt.SequenceNumber
	if d&0x8000 == 0 && (d < 3000 || !outsideRange(b.head.packet.SequenceNumber, pkt.SequenceNumber)) {
		// pkt comes before tail and is within range of head and/or tail, needs to be inserted
		for c := b.tail.prev; c != nil; c = c.prev {
			if (pkt.SequenceNumber-c.packet.SequenceNumber)&0x8000 == 0 {
				// insert after c
				if p.start && outsideRange(c.packet.SequenceNumber, pkt.SequenceNumber) {
					p.reset = true
				} else if pkt.SequenceNumber == c.packet.SequenceNumber+1 {
					if f := pkt.Timestamp - c.packet.Timestamp; f > b.maxSampleSize {
						b.maxSampleSize = f
					}
				}
				c.next.prev = p
				p.next = c.next
				p.prev = c
				c.next = p
				return
			}
		}

		// prepend
		p.reset = p.start && outsideRange(b.prevSN, pkt.SequenceNumber)
		b.head.prev = p
		p.next = b.head
		b.head = p
		return
	}

	// append
	if p.start && outsideRange(b.tail.packet.SequenceNumber, pkt.SequenceNumber) {
		p.reset = true
		b.minTS += b.maxSampleSize
	} else {
		b.minTS += pkt.Timestamp - b.tail.packet.Timestamp
		if pkt.SequenceNumber == b.tail.packet.SequenceNumber+1 {
			if f := pkt.Timestamp - b.tail.packet.Timestamp; f > b.maxSampleSize {
				b.maxSampleSize = f
			}
		}
	}
	p.prev = b.tail
	b.tail.next = p
	b.tail = p
}

func (b *Buffer) Pop(force bool) []*rtp.Packet {
	b.mu.Lock()
	defer b.mu.Unlock()

	if force {
		return b.forcePop()
	} else {
		return b.pop()
	}
}

func (b *Buffer) forcePop() []*rtp.Packet {
	packets := make([]*rtp.Packet, 0, b.size)
	var next *packet
	for c := b.head; c != nil; c = next {
		next = c.next
		packets = append(packets, c.packet)
		b.free(c)
	}
	b.head = nil
	b.tail = nil
	return packets
}

func (b *Buffer) pop() []*rtp.Packet {
	b.drop()

	if b.tail == nil {
		return nil
	}

	prevSN := b.prevSN
	startRequired := true
	var end *packet
	for c := b.head; c != nil; c = c.next {
		// check sequence number
		if c.packet.SequenceNumber != prevSN+1 && (!startRequired || !c.reset) {
			break
		}
		if startRequired {
			if !c.start {
				break
			}
			startRequired = false
		}
		if c.end {
			end = c
			startRequired = true
		}
		prevSN = c.packet.SequenceNumber
	}
	if end == nil {
		return nil
	}

	packets := make([]*rtp.Packet, 0, b.size)
	var next *packet
	for c := b.head; ; c = next {
		next = c.next
		packets = append(packets, c.packet)
		if next != nil {
			if outsideRange(next.packet.SequenceNumber, c.packet.SequenceNumber) {
				// adjust minTS to account for sequence number reset
				b.minTS += next.packet.Timestamp - c.packet.Timestamp - b.maxSampleSize
			}
			next.prev = nil
		}
		if c == end {
			b.prevSN = c.packet.SequenceNumber
			b.head = next
			if next == nil {
				b.tail = nil
			}
			b.free(c)
			return packets
		}
		b.free(c)
	}
}

func (b *Buffer) drop() {
	dropped := false
	for c := b.head; c != nil; {
		if c.packet.Timestamp > b.minTS || (c.packet.Timestamp-b.minTS)&0x80000000 == 0 {
			break
		}

		// drop head until empty or start of next sample
		dropped = true
		for {
			b.logger.Debugw("packet dropped", "sequence number", c.packet.SequenceNumber, "timestamp", c.packet.Timestamp)
			b.head = c.next
			if b.head == nil {
				b.tail = nil
			} else {
				b.head.prev = nil
				if outsideRange(b.head.packet.SequenceNumber, c.packet.SequenceNumber) {
					// adjust minTS to account for sequence number reset
					b.minTS += b.head.packet.Timestamp - c.packet.Timestamp - b.maxSampleSize
				}
			}
			b.free(c)

			c = b.head
			if c == nil {
				break
			} else if c.start {
				b.prevSN = c.packet.SequenceNumber - 1
				break
			}
		}
	}

	if dropped && b.onPacketDropped != nil {
		b.onPacketDropped()
	}
}

func (b *Buffer) newPacket(start, end bool, pkt *rtp.Packet) *packet {
	b.size++
	if b.pool == nil {
		return &packet{
			start:  start,
			end:    end,
			packet: pkt,
		}
	}

	p := b.pool
	b.pool = p.next
	p.next = nil
	p.start = start
	p.end = end
	p.packet = pkt
	return p
}

func (b *Buffer) free(pkt *packet) {
	b.size--
	pkt.prev = nil
	pkt.packet = nil
	pkt.next = b.pool
	b.pool = pkt
}

func outsideRange(a, b uint16) bool {
	return a-b > 3000 && b-a > 3000
}
