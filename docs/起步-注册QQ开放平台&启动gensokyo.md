```markdown
# 创建QQ机器人并配置

首先，您需要在 [QQ开放平台](https://q.qq.com/qqbot/) 注册一个开发者账号，确保使用您的大号QQ进行注册，而非小号。

## 注册步骤

1. 登录 QQ 开放平台，使用大号QQ注册账号。
2. 注册完成后，进入开发者控制台。

## 创建机器人

根据 [图文版教程](https://www.yuque.com/km57bt/hlhnxg/hoxlh53gg11h7r3l) 中的指导操作，创建您的机器人，并进行必要的配置。

## 设置Intent

根据您的频道类型选择合适的Intent设置：

### 私域频道

```yaml
text_intent:
    - DirectMessageHandler
    - CreateMessageHandler
    - InteractionHandler
    - GroupATMessageEventHandler
    - C2CMessageEventHandler
    - GroupMsgRejectHandler
    - GroupMsgReceiveHandler
    - GroupAddRobotEventHandler
    - GroupDelRobotEventHandler
```

### 公域频道

```yaml
text_intent:
    - DirectMessageHandler
    - ATMessageEventHandler
    - InteractionHandler
    - GroupATMessageEventHandler
    - C2CMessageEventHandler
    - GroupMsgRejectHandler
    - GroupMsgReceiveHandler
    - GroupAddRobotEventHandler
    - GroupDelRobotEventHandler
```

确保按照上述格式将Intent配置正确，这将确保机器人能够正确地处理消息和事件。

## 连接nb2和koishi

完成上述基础配置后，您可以继续学习如何使用nb2和koishi等应用程序来开发您的自定义插件。

现在，您已经完成了基础配置和必要的设置，可以开始进行进一步的开发和集成了。
```