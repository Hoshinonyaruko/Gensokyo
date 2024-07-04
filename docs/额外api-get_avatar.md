```markdown
# API: get_avatar

获取用户头像。

## 返回值

```go
type GetAvatarResponse struct {
    Message string      `json:"message"`
    RetCode int         `json:"retcode"`
    Echo    interface{} `json:"echo"`
    UserID  int64       `json:"user_id"`
}
```

## 所需字段

- **group_id**: 群号（当获取群成员头像时需要）
- **user_id**: 用户 QQ 号（当获取私信头像时需要）

## CQcode

CQ头像码格式.支持message segment式传参,将at segment类比修改为avatar即可.
[CQ:avatar,qq=123456]

```