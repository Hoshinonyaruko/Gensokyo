package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo/idmap"
)

func GetIDHandler(c *gin.Context) {
	idOrRow := c.Query("id")
	typeVal, err := strconv.Atoi(c.Query("type"))

	if err != nil || (typeVal != 1 && typeVal != 2) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type"})
		return
	}

	switch typeVal {
	case 1:
		newRow, err := idmap.StoreIDv2(idOrRow)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"row": newRow})

	case 2:
		id, err := idmap.RetrieveRowByIDv2(idOrRow)
		if err == idmap.ErrKeyNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "ID not found"})
			return
		} else if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id})

	case 3:
		// 存储
		section := c.Query("id")
		subtype := c.Query("subtype")
		value := c.Query("value")
		err := idmap.WriteConfigv2(section, subtype, value)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success"})

	case 4:
		// 获取值
		section := c.Query("id")
		subtype := c.Query("subtype")
		value, err := idmap.ReadConfigv2(section, subtype)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"value": value})
	}

}
