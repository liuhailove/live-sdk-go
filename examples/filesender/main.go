package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/liuhailove/live-sdk-go/pkg/samplebuilder"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/h264writer"
	"github.com/pion/webrtc/v3/pkg/media/ivfwriter"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	"os"
	"os/signal"
	"strings"
	"syscall"

	sdk "github.com/liuhailove/live-sdk-go"
)

var (
	host, apiKey, apiSecret, roomName, identity string
)

func init() {
	flag.StringVar(&host, "host", "ws://localhost:7880", "livekit server host")
	flag.StringVar(&apiKey, "api-key", "devkey", "livekit api key")
	flag.StringVar(&apiSecret, "api-secret", "secret", "livekit api secret")
	flag.StringVar(&roomName, "room-name", "测试房间", "room name")
	flag.StringVar(&identity, "identity", "sdk-participant2", "participant identity")
}

func main() {
	flag.Parse()
	if host == "" || apiKey == "" || apiSecret == "" || roomName == "" || identity == "" {
		fmt.Println("invalid arguments.")
		return
	}
	room, err := sdk.ConnectToRoom(host, sdk.ConnectInfo{
		APIKey:              apiKey,
		APISecret:           apiSecret,
		RoomName:            roomName,
		ParticipantIdentity: identity,
	}, &sdk.RoomCallback{
		ParticipantCallback: sdk.ParticipantCallback{
			OnTrackSubscribed: onTrackSubscribed,
		},
	})

	if err != nil {
		panic(err.(any))
	}

	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}

	file := dir + "/examples/filesender/output.h264"
	audioFile := dir + "/examples/filesender/output.ogg"
	videoWidth := 960
	videoHeight := 720
	videoTrack, err2 := sdk.NewLocalFileTrack(
		file,
		sdk.ReaderTrackWithOnWriteComplete(func() {
			fmt.Println("track finished")
		}),
	)
	if err2 != nil {
		panic(err2.(any))
	}
	audioTrack, err3 := sdk.NewLocalFileTrack(audioFile)
	if err3 != nil {
		panic(err3.(any))
	}

	_, err1 := room.LocalParticipant.PublishTrack(videoTrack, &sdk.TrackPublicationOptions{
		Name:        file,
		VideoWidth:  videoWidth,
		VideoHeight: videoHeight,
	})
	room.LocalParticipant.PublishTrack(audioTrack, &sdk.TrackPublicationOptions{
		Name: file,
	})
	if err1 != nil {
		fmt.Println("456", err1.Error())
	}
	////这里必须阻塞一下，否则无法将数据推送出去
	//select {}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	<-sigChan
	room.Disconnect()
}

func onTrackSubscribed(track *webrtc.TrackRemote, publication *sdk.RemoteTrackPublication, rp *sdk.RemoteParticipant) {
	//fileName := fmt.Sprintf("%s-%s", rp.Identity(), track.ID())
	//fmt.Println("write track to file", fileName)
	//NewTrackWriter(track, rp.WritePLI, fileName)
	fmt.Println(track.ID())

}

const (
	maxVideoLate = 1000 // nearly 2s for fhd video
	maxAudioLate = 200  // 4s for audio
)

type TrackWriter struct {
	sb     *samplebuilder.SampleBuilder
	writer media.Writer
	track  *webrtc.TrackRemote
}

func NewTrackWriter(track *webrtc.TrackRemote, pliWriter sdk.PLIWriter, fileName string) (*TrackWriter, error) {
	var (
		sb     *samplebuilder.SampleBuilder
		writer media.Writer
		err    error
	)
	switch {
	case strings.EqualFold(track.Codec().MimeType, "video/vp8"):
		sb = samplebuilder.New(maxVideoLate, &codecs.VP8Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			pliWriter(track.SSRC())
		}))
		// ivfwriter use frame count as PTS, that might cause video played in a incorrect framerate(fast or slow)
		writer, err = ivfwriter.New(fileName + ".ivf")
	case strings.EqualFold(track.Codec().MimeType, "video/h264"):
		sb = samplebuilder.New(maxVideoLate, &codecs.H264Packet{}, track.Codec().ClockRate, samplebuilder.WithPacketDroppedHandler(func() {
			pliWriter(track.SSRC())
		}))
		writer, err = h264writer.New(fileName + ".h264")
	case strings.EqualFold(track.Codec().MimeType, "audio/opus"):
		sb = samplebuilder.New(maxAudioLate, &codecs.OpusPacket{}, track.Codec().ClockRate)
		writer, err = oggwriter.New(fileName+".ogg", 48000, track.Codec().Channels)

	default:
		return nil, errors.New("unsupported codec type")
	}

	if err != nil {
		return nil, err
	}

	t := &TrackWriter{
		sb:     sb,
		writer: writer,
		track:  track,
	}
	go t.start()
	return t, nil
}

func (t *TrackWriter) start() {
	defer t.writer.Close()
	for {
		pkt, _, err := t.track.ReadRTP()
		if err != nil {
			break
		}
		t.sb.Push(pkt)

		for _, p := range t.sb.PopPackets() {
			t.writer.WriteRTP(p)
		}
	}
}
