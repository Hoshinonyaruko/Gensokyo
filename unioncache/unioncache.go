package unioncache

import "sync"

// 双向映射：读用 sync.Map（无锁），写用一个小 mutex 保证两个表原子一致
var (
	id2union sync.Map // map[string]string
	union2id sync.Map // map[string]string

	mu sync.Mutex
)

// Store 建立 1:1 映射：id <-> unionOpenID
// 若 id 或 unionOpenID 原先与别的值绑定，会自动清理旧绑定，保持一对一。
func Store(id, unionOpenID string) {
	if id == "" || unionOpenID == "" {
		// 空 key 会污染映射与导致难排查的问题，直接忽略
		return
	}

	mu.Lock()
	defer mu.Unlock()

	// 1) 如果该 id 之前绑定过别的 union，删掉旧 union->id
	if oldU, ok := id2union.Load(id); ok {
		if u, _ := oldU.(string); u != "" && u != unionOpenID {
			union2id.Delete(u)
		}
	}

	// 2) 如果该 union 之前绑定过别的 id，删掉旧 id->union
	if oldID, ok := union2id.Load(unionOpenID); ok {
		if oid, _ := oldID.(string); oid != "" && oid != id {
			id2union.Delete(oid)
		}
	}

	// 3) 写入新绑定
	id2union.Store(id, unionOpenID)
	union2id.Store(unionOpenID, id)
}

// Union 通过 id 获取 unionOpenID
func Union(id string) (unionOpenID string, ok bool) {
	v, ok := id2union.Load(id)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// ID 通过 unionOpenID 获取 id
func ID(unionOpenID string) (id string, ok bool) {
	v, ok := union2id.Load(unionOpenID)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}

// DeleteID 按 id 删除（会一并删除反向映射）
func DeleteID(id string) {
	if id == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	if u, ok := id2union.Load(id); ok {
		if us, _ := u.(string); us != "" {
			union2id.Delete(us)
		}
	}
	id2union.Delete(id)
}

// DeleteUnion 按 unionOpenID 删除（会一并删除反向映射）
func DeleteUnion(unionOpenID string) {
	if unionOpenID == "" {
		return
	}
	mu.Lock()
	defer mu.Unlock()

	if id, ok := union2id.Load(unionOpenID); ok {
		if ids, _ := id.(string); ids != "" {
			id2union.Delete(ids)
		}
	}
	union2id.Delete(unionOpenID)
}

// Clear 清空（建议仅用于测试或受控重置）
func Clear() {
	mu.Lock()
	defer mu.Unlock()
	id2union = sync.Map{}
	union2id = sync.Map{}
}
