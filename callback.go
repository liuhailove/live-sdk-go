package live_sdk_go

import (
	"github.com/livekit/protocol/livekit"
	"github.com/pion/webrtc/v3"
)

type ParticipantCallback struct {
	// for all participants
	// 音轨静音回调
	OnTrackMuted func(pub TrackPublication, p Participant)
	// 解除静音回调
	OnTrackUnmuted func(pub TrackPublication, p Participant)
	// 原数据变更回调
	OnMetadataChanged func(oldMetadata string, p Participant)
	// 是否为发言者变更回调
	OnIsSpeakingChanged func(p Participant)
	// 连接质量变更回调
	OnConnectionQualityChanged func(update *livekit.ConnectionQualityInfo, p Participant)

	// for remote participants
	// 音轨订阅通知
	OnTrackSubscribed func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant)
	// 解除订阅通知
	OnTrackUnsubscribed func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant)
	// 订阅失败通知
	OnTrackSubscriptionFailed func(sid string, rp *RemoteParticipant)
	// 音轨发布通知
	OnTrackPublished func(publication *RemoteTrackPublication, rp *RemoteParticipant)
	// 音轨未发布通知
	OnTrackUnpublished func(publication *RemoteTrackPublication, rp *RemoteParticipant)
	// 收到数据通知
	OnDataReceived func(data []byte, rp *RemoteParticipant)
}

func NewParticipantCallback() *ParticipantCallback {
	return &ParticipantCallback{
		OnTrackMuted:               func(pub TrackPublication, p Participant) {},
		OnTrackUnmuted:             func(pub TrackPublication, p Participant) {},
		OnMetadataChanged:          func(oldMetadata string, p Participant) {},
		OnIsSpeakingChanged:        func(p Participant) {},
		OnConnectionQualityChanged: func(update *livekit.ConnectionQualityInfo, p Participant) {},
		OnTrackSubscribed:          func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant) {},
		OnTrackUnsubscribed:        func(track *webrtc.TrackRemote, publication *RemoteTrackPublication, rp *RemoteParticipant) {},
		OnTrackSubscriptionFailed:  func(sid string, rp *RemoteParticipant) {},
		OnTrackPublished:           func(publication *RemoteTrackPublication, rp *RemoteParticipant) {},
		OnTrackUnpublished:         func(publication *RemoteTrackPublication, rp *RemoteParticipant) {},
		OnDataReceived:             func(data []byte, rp *RemoteParticipant) {},
	}
}
func (cb *ParticipantCallback) Merge(other *ParticipantCallback) {
	if other.OnTrackMuted != nil {
		cb.OnTrackMuted = other.OnTrackMuted
	}
	if other.OnTrackUnmuted != nil {
		cb.OnTrackUnmuted = other.OnTrackUnmuted
	}
	if other.OnMetadataChanged != nil {
		cb.OnMetadataChanged = other.OnMetadataChanged
	}
	if other.OnIsSpeakingChanged != nil {
		cb.OnIsSpeakingChanged = other.OnIsSpeakingChanged
	}
	if other.OnConnectionQualityChanged != nil {
		cb.OnConnectionQualityChanged = other.OnConnectionQualityChanged
	}
	if other.OnTrackSubscribed != nil {
		cb.OnTrackSubscribed = other.OnTrackSubscribed
	}
	if other.OnTrackUnsubscribed != nil {
		cb.OnTrackUnsubscribed = other.OnTrackUnsubscribed
	}
	if other.OnTrackSubscriptionFailed != nil {
		cb.OnTrackSubscriptionFailed = other.OnTrackSubscriptionFailed
	}
	if other.OnTrackPublished != nil {
		cb.OnTrackPublished = other.OnTrackPublished
	}
	if other.OnTrackUnpublished != nil {
		cb.OnTrackUnpublished = other.OnTrackUnpublished
	}
	if other.OnDataReceived != nil {
		cb.OnDataReceived = other.OnDataReceived
	}
}

type RoomCallback struct {
	OnDisconnected            func()
	OnParticipantConnected    func(*RemoteParticipant)
	OnParticipantDisconnected func(*RemoteParticipant)
	OnActiveSpeakersChanged   func([]Participant)
	OnRoomMetadataChanged     func(metadata string)
	OnReconnecting            func()
	OnReconnected             func()

	// participant events are sent to the room as well
	ParticipantCallback
}

func NewRoomCallback() *RoomCallback {
	pc := NewParticipantCallback()
	return &RoomCallback{
		ParticipantCallback: *pc,

		OnDisconnected:            func() {},
		OnParticipantConnected:    func(participant *RemoteParticipant) {},
		OnParticipantDisconnected: func(participant *RemoteParticipant) {},
		OnActiveSpeakersChanged:   func(participants []Participant) {},
		OnRoomMetadataChanged:     func(metadata string) {},
		OnReconnecting:            func() {},
		OnReconnected:             func() {},
	}
}

func (cb *RoomCallback) Merge(other *RoomCallback) {
	if other == nil {
		return
	}

	if other.OnDisconnected != nil {
		cb.OnDisconnected = other.OnDisconnected
	}
	if other.OnParticipantConnected != nil {
		cb.OnParticipantConnected = other.OnParticipantConnected
	}
	if other.OnParticipantDisconnected != nil {
		cb.OnParticipantDisconnected = other.OnParticipantDisconnected
	}
	if other.OnActiveSpeakersChanged != nil {
		cb.OnActiveSpeakersChanged = other.OnActiveSpeakersChanged
	}
	if other.OnRoomMetadataChanged != nil {
		cb.OnRoomMetadataChanged = other.OnRoomMetadataChanged
	}
	if other.OnReconnecting != nil {
		cb.OnReconnecting = other.OnReconnecting
	}
	if other.OnReconnected != nil {
		cb.OnReconnected = other.OnReconnected
	}
	cb.ParticipantCallback.Merge(&other.ParticipantCallback)
}
