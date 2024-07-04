```markdown
## md cq码/segment格式

### CQ码格式

```markdown
[CQ:markdown,data=xxx]
```

推荐使用模板：[链接到模板](https://github.com/hoshinonyaruko/gensokyo-qqmd)

- `data,xxx` 是经过base64编码的JSON数据，支持与其他CQ码拼接，可以组合成message segment形式。

官方文档：[开发者文档](https://bot.q.qq.com/wiki/develop)

新文档：[API v2文档](https://bot.q.qq.com/wiki/develop/api-v2/)

### 自定义md格式

```json
{
  "markdown": {
    "content": "你好"
  },
  "keyboard": {
    "content": {
      "rows": [
        {
          "buttons": [
            {
              "render_data": {
                "label": "再来一张",
                "visited_label": "正在绘图",
                "style": 1
              },
              "action": {
                "type": 2,
                "permission": {
                  "type": 2,
                  "specify_role_ids": [
                    "1",
                    "2",
                    "3"
                  ]
                },
                "click_limit": 10,
                "unsupport_tips": "编辑-兼容文本",
                "data": "你好",
                "at_bot_show_channel_list": false
              }
            }
          ]
        }
      ]
    }
  },
  "msg_id": "123",
  "timestamp": "123",
  "msg_type": 2
}
```

### 模板md格式

```json
{
  "markdown": {
    "custom_template_id": "101993071_1658748972",
    "params": [
      {
        "key": "text",
        "values": ["标题"]
      },
      {
        "key": "image",
        "values": [
          "https://resource5-1255303497.cos.ap-guangzhou.myqcloud.com/abcmouse_word_watch/other/mkd_img.png"
        ]
      }
    ]
  },
  "keyboard": {
    "content": {
      "rows": [
        {
          "buttons": [
            {
              "render_data": {
                "label": "再来一次",
                "visited_label": "再来一次"
              },
              "action": {
                "type": 1,
                "permission": {
                  "type": 1,
                  "specify_role_ids": [
                    "1",
                    "2",
                    "3"
                  ]
                },
                "click_limit": 10,
                "unsupport_tips": "兼容文本",
                "data": "data",
                "at_bot_show_channel_list": true
              }
            }
          ]
        }
      ]
    }
  }
}
```

### 按钮格式

```json
{
  "keyboard": {
    "id": 1,
    "rows": [
      {
        "buttons": [
          {
            "render_data": {
              "label": "再来一次",
              "visited_label": "再来一次"
            },
            "action": {
              "type": 1,
              "permission": {
                "type": 1,
                "specify_role_ids": [
                  "1",
                  "2",
                  "3"
                ]
              },
              "click_limit": 10,
              "unsupport_tips": "兼容文本",
              "data": "data",
              "at_bot_show_channel_list": true
            }
          }
        ]
      }
    ]
  }
}
```

### 图文混排格式

```markdown
{{.text}}![{{.image_info}}]({{.image_url}})
```

![{{.image_info}}]({{.image_url}}){{.text}}

注意：在`{{}}`中不可以使用`![]()`这种Markdown格式的关键字。

![text #208px #320px](https://xxxxx.png)
```

详细文档请参考：[发消息含有消息按钮组件的消息](https://bot.q.qq.com/wiki/develop/api/openapi/message/post_keyboard_messages.html#%E5%8F%91%E9%80%81%E5%90%AB%E6%9C%89%E6%B6%88%E6%81%AF%E6%8C%89%E9%92%AE%E7%BB%84%E4%BB%B6%E7%9A%84%E6%B6%88%E6%81%AF)
```