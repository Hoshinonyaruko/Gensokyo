package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
)

func init() {
	callapi.RegisterHandler("send_msg_async", HandleSendMsg)
}
