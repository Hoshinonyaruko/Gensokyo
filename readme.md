<p align="center">
  <a href="https://www.github.com/hoshinonyaruko/gensokyo">
    <img src="images/head.gif" width="200" height="200" alt="gensokyo">
  </a>
</p>

<div align="center">

# gensokyo

_✨ 基于 [OneBot](https://github.com/howmanybots/onebot/blob/master/README.md) QQ官方机器人Api Golang 原生实现 ✨_  


</div>

<p align="center">
  <a href="https://raw.githubusercontent.com/hoshinonyaruko/gensokyo/main/LICENSE">
    <img src="https://img.shields.io/github/license/hoshinonyaruko/gensokyo" alt="license">
  </a>
  <a href="https://github.com/hoshinonyaruko/gensokyo/releases">
    <img src="https://img.shields.io/github/v/release/hoshinonyaruko/gensokyo?color=blueviolet&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/howmanybots/onebot/blob/master/README.md">
    <img src="https://img.shields.io/badge/OneBot-v11-blue?style=flat&logo=data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAEAAAABABAMAAABYR2ztAAAAIVBMVEUAAAAAAAADAwMHBwceHh4UFBQNDQ0ZGRkoKCgvLy8iIiLWSdWYAAAAAXRSTlMAQObYZgAAAQVJREFUSMftlM0RgjAQhV+0ATYK6i1Xb+iMd0qgBEqgBEuwBOxU2QDKsjvojQPvkJ/ZL5sXkgWrFirK4MibYUdE3OR2nEpuKz1/q8CdNxNQgthZCXYVLjyoDQftaKuniHHWRnPh2GCUetR2/9HsMAXyUT4/3UHwtQT2AggSCGKeSAsFnxBIOuAggdh3AKTL7pDuCyABcMb0aQP7aM4AnAbc/wHwA5D2wDHTTe56gIIOUA/4YYV2e1sg713PXdZJAuncdZMAGkAukU9OAn40O849+0ornPwT93rphWF0mgAbauUrEOthlX8Zu7P5A6kZyKCJy75hhw1Mgr9RAUvX7A3csGqZegEdniCx30c3agAAAABJRU5ErkJggg==" alt="gensokyo">
  </a>
  <a href="https://github.com/hoshinonyaruko/gensokyo/actions">
    <img src="images/badge.svg" alt="action">
  </a>
  <a href="https://goreportcard.com/report/github.com/hoshinonyaruko/gensokyo">
  <img src="https://goreportcard.com/badge/github.com/hoshinonyaruko/gensokyo" alt="GoReportCard">
  </a>
</p>

<p align="center">
  <a href="https://github.com/howmanybots/onebot/blob/master/README.md">文档</a>
  ·
  <a href="https://github.com/hoshinonyaruko/gensokyo/releases">下载</a>
  ·
  <a href="https://github.com/hoshinonyaruko/gensokyo/releases">开始使用</a>
  ·
  <a href="https://github.com/hoshinonyaruko/gensokyo/blob/master/CONTRIBUTING.md">参与贡献</a>
</p>
<p align="center">
  <a href="https://gensokyo.bot">项目主页:gensokyo.bot</a>
</p>

## 引用
- [`tencent-connect/botgo`](https://github.com/tencent-connect/botgo): 本项目引用了此项目,并做了一些改动.

## 介绍
gensokyo兼容 [OneBot-v11](https://github.com/botuniverse/onebot-11) ，并在其基础上做了一些扩展，详情请看 OneBot 的文档。

Gensokyo文档(施工中):[起步](/docs/起步-注册QQ开放平台&启动gensokyo.md)

可将官方的websocket和api转换至onebotv11标准,

支持连接koishi,nonebot2,trss,zerobot,MiraiCQ,hoshino..

支持连接tata,派蒙,炸毛,早苗,yobot...

支持连接Mirai(Overflow)...

可以与支持onebotV11适配器的项目相连接使用.

实现插件开发和用户开发者无需重新开发,复用过往生态的插件和使用体验.

持续完善中.....交流群:196173384

欢迎测试,询问任何有关使用的问题,有问必答,有难必帮~

[Gensokyo临时文档](https://www.yuque.com/km57bt/hlhnxg/mw7gm8dlpccd324e)展开左侧折叠栏,临时文档包含markdown定义、额外api文档等内容

后续会将文档独立，因为语雀文档公开查看无需登录需要vip，故暂时放在我的机器人文档中。临时文档也包含了Gensokyo的完整编译教程。

## 特别鸣谢

- [`mnixry/nonebot-plugin-gocqhttp`](https://github.com/mnixry/nonebot-plugin-gocqhttp/): 本项目采用了mnixry编写的前端,并实现了与它对应的,基于qq官方api的后端api.
- 特别鸣谢[`dk 盾`](https://www.dkdun.cn/),友情赞助服务器资源

### 接口

- [x] HTTP API
- [x] 反向 HTTP POST
- [x] 正向 WebSocket
- [x] 反向 WebSocket

### 拓展支持

> 拓展 API 可前往 [文档](docs/cqhttp.md) 查看

- [x] 连接多个ws地址
- [x] 将频道虚拟成群事件
- [x] 将私信虚拟成频道或群事件
- [x] webui,可以在webui修改配置,查看频道列表,发送信息
- [x] 方便过审的指令黑白名单
- [x] 自动url转换(自备域名)
- [x] 可自定义图片压缩\图床服务
- [x] 可编辑的数据库
- [x] 支持array和信息段
- [x] 文字,图片,语音,视频,MD,支持多种类型发送
- [x] 支持全域,频道,频道私聊,群,群私聊
- [x] 主动信息失败自动转被动,提高信息传达可靠性
- [x] 提前于官方支持群列表 群成员 api
- [x] 完善的重连,健壮的连接能力.
- [x] 支持[CQ:markdown,data=] Markdown发送
- [x] [`markdown文档`](https://www.yuque.com/km57bt/hlhnxg/ddkv4a2lgcswitei)
- [x] 持续更新~


### 实现

<details>
<summary>已实现 CQ 码</summary>

#### 符合 OneBot 标准的 CQ 码

| CQ 码        | 功能                        |
| ------------ | --------------------------- |
| [CQ:face]    | [QQ 表情]                   |
| [CQ:record]  | [语音]                      |
| [CQ:video]   | [短视频]                    |
| [CQ:at]      | [@某人]                     |
| [CQ:share]   | [链接分享]                  |
| [CQ:music]   | [音乐分享] [音乐自定义分享] |
| [CQ:reply]   | [回复]                      |
| [CQ:forward] | [合并转发]                  |
| [CQ:node]    | [合并转发节点]              |
| [CQ:xml]     | [XML 消息]                  |
| [CQ:json]    | [JSON 消息]                 |

todo,正在施工中

#### 拓展 CQ 码及与 OneBot 标准有略微差异的 CQ 码

| 拓展 CQ 码     | 功能                              |
| -------------- | --------------------------------- |
| [CQ:image]     | [图片]                            |
| [CQ:poke]      | [戳一戳]                          |
| [CQ:node]      | [合并转发消息节点]                |
| [CQ:markdown]  | [markdown卡片收发] |
| [CQ:tts]       | [文本转语音]                      |


</details>

<details>
<summary>已实现 API</summary>

#### 符合 OneBot 标准的 API

| API                      | 功能                   |
| ------------------------ | ---------------------- |
| /send_private_msg√        | [发送私聊消息]         |
| /send_group_msg√         | [发送群消息]           |
| /send_guild_channel_msg√ | [发送频道消息]         |
| /send_msg√               | [发送消息]             |
| /delete_msg              | [撤回信息]             |
| /set_group_kick          | [群组踢人]             |
| /set_group_ban√          | [群组单人禁言]         |
| /set_group_whole_ban√    | [群组全员禁言]         |
| /set_group_admin         | [群组设置管理员]       |
| /set_group_card          | [设置群名片（群备注）] |
| /set_group_name          | [设置群名]             |
| /set_group_leave         | [退出群组]             |
| /set_group_special_title | [设置群组专属头衔]     |
| /set_friend_add_request  | [处理加好友请求]       |
| /set_group_add_request   | [处理加群请求/邀请]    |
| /get_login_info√         | [获取登录号信息]       |
| /get_stranger_info       | [获取陌生人信息]       |
| /get_friend_list√        | [获取好友列表]         |
| /get_group_info√          | [获取群/频道信息]     |
| /get_group_list√         | [获取群列表]           |
| /get_group_member_info√  | [获取群成员信息]       |
| /get_group_member_list√  | [获取群成员列表]       |
| /get_group_honor_info    | [获取群荣誉信息]       |
| /can_send_image√         | [检查是否可以发送图片] |
| /can_send_record         | [检查是否可以发送语音] |
| /get_version_info√       | [获取版本信息]         |
| /set_restart√             | [重启 gensokyo]       |
| /.handle_quick_operation | [对事件执行快速操作]   |


#### 拓展 API 及与 OneBot 标准有略微差异的 API

| 拓展 API                    | 功能                   |
| --------------------------- | ---------------------- |
| /set_group_portrait         | [设置群头像]           |
| /get_image                  | [获取图片信息]         |
| /get_msg                    | [获取消息]             |
| /get_forward_msg            | [获取合并转发内容]     |
| /send_group_forward_msg√     | [发送合并转发(群)]     |
| /.get_word_slices           | [获取中文分词]         |
| /.ocr_image                 | [图片 OCR]             |
| /get_group_system_msg       | [获取群系统消息]       |
| /get_group_file_system_info | [获取群文件系统信息]   |
| /get_group_root_files       | [获取群根目录文件列表] |
| /get_group_files_by_folder  | [获取群子目录文件列表] |
| /get_group_file_url         | [获取群文件资源链接]   |
| /get_status√                 | [获取状态]             |


</details>

<details>
<summary>已实现 Event</summary>

#### 符合 OneBot 标准的 Event（部分 Event 比 OneBot 标准多上报几个字段，不影响使用）

| 事件类型 | Event            |
| -------- | ---------------- |
| 消息事件 | [私聊信息]√       |
| 消息事件 | [群消息]√         |
| 通知事件 | [群文件上传]     |
| 通知事件 | [群管理员变动]   |
| 通知事件 | [群成员减少]     |
| 通知事件 | [群成员增加]     |
| 通知事件 | [群禁言]         |
| 通知事件 | [好友添加]       |
| 通知事件 | [群消息撤回]     |
| 通知事件 | [好友消息撤回]   |
| 通知事件 | [群内戳一戳]     |
| 通知事件 | [群红包运气王]   |
| 通知事件 | [群成员荣誉变更] |
| 请求事件 | [加好友请求]     |
| 请求事件 | [加群请求/邀请]  |


#### 拓展 Event

| 事件类型 | 拓展 Event       |
| -------- | ---------------- |
| 通知事件 | [好友戳一戳]     |
| 通知事件 | [群内戳一戳]     |
| 通知事件 | [群成员名片更新] |
| 通知事件 | [接收到离线文件] |


</details>

## 关于 ISSUE

以下 ISSUE 会被直接关闭

- 提交 BUG 不使用 Template
- 询问已知问题
- 提问找不到重点
- 重复提问

> 请注意, 开发者并没有义务回复您的问题. 您应该具备基本的提问技巧。  
> 有关如何提问，请阅读[《提问的智慧》](https://github.com/ryanhanwu/How-To-Ask-Questions-The-Smart-Way/blob/main/README-zh_CN.md)

## 性能

10mb内存占用 端口错开可多开 稳定运行无报错