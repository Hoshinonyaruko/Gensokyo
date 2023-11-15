package handlers

import (
	"github.com/hoshinonyaruko/gensokyo/callapi"
)

func init() {
	callapi.RegisterHandler(".handle_quick_operation_async", handle_quick_operation)
}
