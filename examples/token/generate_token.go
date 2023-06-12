package main

import (
	"fmt"
	"github.com/livekit/protocol/auth"
	"time"
)

func main() {
	//用于连接livekit服务器的认证密钥，livekit.yaml中获取
	apiKey := "devkey"
	apiSecret := "secret"
	canPublish := true
	canSubscribe := true
	//生成认证实体
	grant := auth.NewAccessToken(apiKey, apiSecret).AddGrant(&auth.VideoGrant{
		RoomJoin:     true,
		Room:         "测试房间",
		CanPublish:   &canPublish,
		CanSubscribe: &canSubscribe,
	})
	//设置实体对象
	jwt, err := grant.SetIdentity("mytest").SetValidFor(time.Hour).ToJWT()
	if err != nil {

	}
	fmt.Println(jwt)
}
