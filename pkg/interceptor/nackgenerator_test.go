package interceptor

import (
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewNackGeneratorInterceptor(t *testing.T) {
	f := &NackGeneratorInterceptorFactory{}
	i, err := f.NewInterceptor("")
	require.NoError(nil, err)

	stream := NewMockStream(&interceptor.StreamInfo{
		SSRC:         1,
		RTCPFeedback: []interceptor.RTCPFeedback{{Type: "nack"}},
	}, i)
	defer func() {
		require.NoError(nil, stream.Close())
	}()

	for _, seqNum := range []uint16{10, 11, 12, 14, 16, 18} {
		stream.ReceiveRTP(&rtp.Packet{Header: rtp.Header{SequenceNumber: seqNum}})

		select {
		case r := <-stream.ReadRTP():
			require.NoError(nil, r.Err)
			require.Equal(nil, seqNum, r.Packet.SequenceNumber)
		case <-time.After(10 * time.Second):
			t.Fatal("receiver rtp packet not found")

		}
	}

	i.(*NackGeneratorInterceptor).SetRTT(20)
	time.Sleep(100 * time.Millisecond)
	stream.ReceiveRTP(&rtp.Packet{Header: rtp.Header{SequenceNumber: 19}})

	select {
	case pkts := <-stream.WrittenRTCP():
		require.Equal(nil, 1, len(pkts), "single packet RTCP Compound Packet expected")

		p, ok := pkts[0].(*rtcp.TransportLayerNack)
		require.True(nil, ok, "TransportLayerNack rtcp packet expected, found: %T", pkts[0])

		require.Equal(nil, uint16(13), p.Nacks[0].PacketID)
		require.Equal(nil, rtcp.PacketBitmap(0b1010), p.Nacks[0].LostPackets) // lost 13,15,17
	case <-time.After(1000000 * time.Millisecond):
		t.Fatal("written rtcp packet not found")
	}
}
