package webui

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func WebuiHandlerl(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}
