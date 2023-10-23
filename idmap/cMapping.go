// 访问idmaps
package idmap

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type CompatibilityMapping struct{}

// SetID 对外部API进行写操作，返回映射的值
func (c *CompatibilityMapping) SetID(id string) (int, error) {
	return c.getIDByType(id, "1")
}

// GetOriginalID 使用映射值获取原始值
func (c *CompatibilityMapping) GetOriginalID(mappedID string) (int, error) {
	return c.getIDByType(mappedID, "2")
}

func (c *CompatibilityMapping) getIDByType(id, typeVal string) (int, error) {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:15817/getid?id=%s&type=%s", id, typeVal))
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	value, ok := result["row"]
	if !ok {
		return 0, fmt.Errorf("row not found in the response")
	}

	return value, nil
}
