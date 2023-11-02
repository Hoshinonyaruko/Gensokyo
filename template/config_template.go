package template

const ConfigTemplate = `
version: 1
settings:
  ws_address: ["ws://<YOUR_WS_ADDRESS>:<YOUR_WS_PORT>"] # WebSocket服务的地址 支持多个["","",""]
  app_id: 12345                             # 你的应用ID
  token: "<YOUR_APP_TOKEN>"                          # 你的应用令牌
  client_secret: "<YOUR_CLIENT_SECRET>"              # 你的客户端密钥

  text_intent:                                       # 请根据公域 私域来选择intent,错误的intent将连接失败
    - "ATMessageEventHandler"                        # 频道at信息
    - "DirectMessageHandler"                         # 私域频道私信(dms)
    # - "ReadyHandler"                               # 连接成功
    # - "ErrorNotifyHandler"                         # 连接关闭
    # - "GuildEventHandler"                          # 频道事件
    # - "MemberEventHandler"                         # 频道成员新增
    # - "ChannelEventHandler"                        # 频道事件
    # - "CreateMessageHandler"                       # 频道不at信息 私域机器人需要开启 公域机器人开启会连接失败
    # - "InteractionHandler"                         # 添加频道互动回应
    # - "GroupATMessageEventHandler"                 # 群at信息 仅频道机器人时候需要注释
    # - "C2CMessageEventHandler"                     # 群私聊 仅频道机器人时候需要注释
    # - "ThreadEventHandler"                         # 发帖事件 (当前版本已禁用)


  global_channel_to_group: true                      # 是否将频道转换成群 默认true
  global_private_to_channel: false                   # 是否将私聊转换成频道 如果是群场景 会将私聊转为群(方便提审\测试)
  array: false


  server_dir: "<YOUR_SERVER_DIR>" # 提供图片上传服务的服务器(图床)需要带端口号. 如果需要发base64图,需为公网ip,且开放对应端口
  port: "15630"                                # idmaps和图床对外开放的端口号


  lotus: false                                       # lotus特性默认为false,当为true时,将会连接到另一个lotus为false的gensokyo。
                                                     # 使用它提供的图床和idmaps服务(场景:同一个机器人在不同服务器运行,或内网需要发送base64图)。
                                                     # 如果需要发送base64图片,需要设置正确的公网server_dir和开放对应的port


  ws_token: ["","",""]      #连接wss地址时服务器所需的token,如果是ws,可留空,按顺序一一对应
  master_id : ["1","2"]     #群场景尚未开放获取管理员和列表能力,手动从日志中获取需要设置为管理,的user_id并填入(适用插件有权限判断场景)
  enable_ws_server: true    #是否启用正向ws服务器 监听server_dir:port/ws
  ws_server_token : "12345" #正向ws的token 不启动正向ws可忽略
  identify_file: true  #自动生成域名校验文件,在q.qq.com配置信息URL,在server_dir填入自己已备案域名,正确解析到机器人所在服务器ip地址,机器人即可发送链接
  crt: "" #证书路径 从你的域名服务商或云服务商申请签发SSL证书(qq要求SSL) 
  key: "" #密钥路径 Apache（crt文件、key文件）示例: "C:\\123.key" \需要双写成\\
  developer_log : false    #开启开发者日志 默认关闭
  server_user_name : "useradmin"    #默认网页面板用户名
  server_user_password : "admin"    #默认网页面板密码
`
const Logo = `
'                                                                                                      
'    ,hakurei,                                                      ka                                  
'   ho"'     iki                                                    gu                                  
'  ra'                                                              ya                                  
'  is              ,kochiya,    ,sanae,    ,Remilia,   ,Scarlet,    fl   and  yu        ya   ,Flandre,   
'  an      Reimu  'Dai   sei  yas     aka  Rei    sen  Ten     shi  re  sca    yu      ku'  ta"     "ko  
'  Jun        ko  Kirisame""  ka       na    Izayoi,   sa       ig  Koishi       ko   mo'   ta       ga  
'   you.     rei  sui   riya  ko       hi  Ina    baI  'ran   you   ka  rlet      komei'    "ra,   ,sa"  
'     "Marisa"      Suwako    ji       na   "Sakuya"'   "Cirno"'    bu     sen     yu''        Satori  
'                                                                                ka'                   
'                                                                               ri'                    
`
