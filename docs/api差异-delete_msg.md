# API: delete_msg

撤回消息。

## 参数

| 字段名      | 数据类型       | 默认值 | 说明                              |
|-------------|----------------|--------|-----------------------------------|
| message_id  | number (int32) | -      | 消息 ID                           |
| user_id     | number         | -      | 对方 QQ 号（消息类型为 private 时需要） |
| group_id    | number         | -      | 群号（消息类型为 group 时需要）       |
| channel_id  | number         | -      | 频道号（消息类型是 guild 时需要）     |
| guild_id    | number         | -      | 子频道号（消息类型是 guild_Private 时需要） |

## 响应数据

无