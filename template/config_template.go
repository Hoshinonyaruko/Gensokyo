package template

const ConfigTemplate = `
version: 1
settings:
  #反向ws设置
  ws_address: ["ws://<YOUR_WS_ADDRESS>:<YOUR_WS_PORT>"] # WebSocket服务的地址 支持多个["","",""]
  ws_token: ["","",""]              #连接wss地址时服务器所需的token,按顺序一一对应,如果是ws地址,没有密钥,请留空.
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
  array: false                                       # 连接trss云崽请开启array
  hash_id : false                                    # 使用hash来进行idmaps转换,可以让user_id不是123开始的递增值

  server_dir: "<YOUR_SERVER_DIR>"                    # 提供图片上传服务的服务器(图床)需要带端口号. 如果需要发base64图,需为公网ip,且开放对应端口
  port: "15630"                                      # idmaps和图床对外开放的端口号
  backup_port : "5200"                               # 当totus为ture时,port值不再是本地webui的端口,使用lotus_Port来访问webui

  lotus: false                                       # lotus特性默认为false,当为true时,将会连接到另一个lotus为false的gensokyo。
                                                     # 使用它提供的图床和idmaps服务(场景:同一个机器人在不同服务器运行,或内网需要发送base64图)。
                                                     # 如果需要发送base64图片,需要设置正确的公网server_dir和开放对应的port
  lotus_password : ""                                # lotus鉴权 设置后,从gsk需要保持相同密码来访问主gsk

  #增强配置项                                           

  image_sizelimit : 0               #代表kb 腾讯api要求图片1500ms完成传输 如果图片发不出 请提升上行或设置此值 默认为0 不压缩
  image_limit : 100                 #每分钟上传的最大图片数量,可自行增加
  master_id : ["1","2"]             #群场景尚未开放获取管理员和列表能力,手动从日志中获取需要设置为管理,的user_id并填入(适用插件有权限判断场景)
  record_sampleRate : 24000         #语音文件的采样率 最高48000 默认24000 单位Khz
  record_bitRate : 24000            #语音文件的比特率 默认25000 代表 25 kbps 最高无限 请根据带宽 您发送的实际码率调整
  card_nick : ""                    #默认为空,连接mirai-overflow时,请设置为非空,这里是机器人对用户称谓,为空为插件获取,mirai不支持
  auto_bind : true                  #测试功能,后期会移除
  AMsgRetryAsPMsg_Count : 1         #当主动信息发送失败时,自动转为后续的被动信息发送,需要开启Lazy message id,该配置项为每次跟随被动信息发送的信息数量,最大5,建议1-3
  reconnect_times : 100             #反向ws连接失败后的重试次数,希望一直重试,可设置9999
  heart_beat_interval : 10          #反向ws心跳间隔 单位秒 推荐5-10
  launch_reconnect_times : 1        #启动时尝试反向ws连接次数,建议先打开应用端再开启gensokyo,因为启动时连接会阻塞webui启动,默认只连接一次,可自行增大
  native_ob11 : false               #如果你的机器人收到事件报错,请开启此选项增加兼容性
  ramdom_seq : false                #当多开gensokyo时,如果遇到群信息只能发出一条,请开启每个gsk的此项.(建议使用一个gsk连接多个应用)
  url_to_qrimage : false            #将信息中的url转换为二维码单独作为图片发出,需要同时设置  #SSL配置类 机器人发送URL设置 的 transfer_url 为 true visible_ip也需要为true
  qr_size : 200                     #二维码尺寸,单位像素
  guild_url_image_to_base64 : false #解决频道发不了某些url图片,报错40003问题
  oss_type : 0                      #请完善后方具体配置 完成#腾讯云配置...,0代表配置server dir port服务器自行上传(省钱),1,腾讯cos存储桶 2,百度oss存储桶 3,阿里oss存储桶
  self_introduce : ["",""]          #自我介绍,可设置多个随机发送,当不为空时,机器人被邀入群会发送自定义自我介绍 需手动添加新textintent   - "GroupAddRobotEventHandler"   - "GroupDelRobotEventHandler"

  #正向ws设置
  ws_server_path : "ws"             #默认监听0.0.0.0:port/ws_server_path 若有安全需求,可不放通port到公网,或设置ws_server_token 若想监听/ 可改为"",若想监听到不带/地址请写nil
  enable_ws_server: true            #是否启用正向ws服务器 监听server_dir:port/ws_server_path
  ws_server_token : "12345"         #正向ws的token 不启动正向ws可忽略 可为空

  #SSL配置类 机器人发送URL设置

  identify_file : true               #自动生成域名校验文件,在q.qq.com配置信息URL,在server_dir填入自己已备案域名,正确解析到机器人所在服务器ip地址,机器人即可发送链接
  identify_appids : []               #默认不需要设置,完成SSL配置类+server_dir设置为域名+完成备案+ssl全套设置后,若有多个机器人需要过域名校验(自己名下)可设置,格式为,整数appid,组成的数组
  crt : ""                           #证书路径 从你的域名服务商或云服务商申请签发SSL证书(qq要求SSL) 
  key : ""                           #密钥路径 Apache（crt文件、key文件）示例: "C:\\123.key" \需要双写成\\
  transfer_url : true                #默认开启,关闭后自理url发送,配置server_dir为你的域名,配置crt和key后,将域名/url和/image在q.qq.com后台通过校验,自动使用302跳转处理机器人发出的所有域名.

  #日志类

  developer_log : false             #开启开发者日志 默认关闭
  log_level : 1                     # 0=debug 1=info 2=warning 3=error 默认1
  save_logs : false                 #自动储存日志

  #webui设置

  server_user_name : "useradmin"    #默认网页面板用户名
  server_user_password : "admin"    #默认网页面板密码

  #指令过滤类

  remove_prefix : false             #是否忽略公域机器人指令前第一个/
  remove_at : false                 #是否忽略公域机器人指令前第一个[CQ:aq,qq=机器人] 场景(公域机器人,但插件未适配at开头)
  remove_bot_at_group : true        #因为群聊机器人不支持发at,开启本开关会自动隐藏群机器人发出的at(不影响频道场景)
  add_at_group : false              #自动在群聊指令前加上at,某些机器人写法特别,必须有at才反应时,请打开,默认请关闭(如果需要at,不需要at指令混杂,请优化代码适配群场景,群场景目前没有at概念)

  white_prefix_mode : false         #公域 过审用 指令白名单模式开关 如果审核严格 请开启并设置白名单指令 以白名单开头的指令会被通过,反之被拦截
  white_prefixs : [""]              #可设置多个 比如设置 机器人 测试 则只有信息以机器人 测试开头会相应 remove_prefix remove_at 需为true时生效
  white_bypass : []                 #格式[1,2,3],白名单不生效的群或用户(私聊时),用于设置自己的灰度沙箱群/灰度沙箱私聊,避免开发测试时反复开关白名单的不便,请勿用于生产环境.
  white_enable : [true,true,true,true,true] #指令白名单生效范围,5个分别对应,频道公(ATMessageEventHandler),频道私(CreateMessageHandler),频道私聊,群,群私聊,改成false,这个范围就不生效指令白名单(使用场景:群全量,频道私域的机器人,或有私信资质的机器人)
  white_bypass_reverse : false      #反转white_bypass的效果,可仅在white_bypass应用white_prefix_mode,场景:您的不同用户群,可以开放不同层次功能,便于您的运营和规化(测试/正式环境)
  No_White_Response : ""            #默认不兜底,强烈建议设置一个友善的兜底回复,告知审核机器人已无隐藏指令,如:你输入的指令不对哦,@机器人来获取可用指令

  black_prefix_mode : false         #公私域 过审用 指令黑名单模式开关 过滤被审核打回的指令不响应 无需改机器人后端
  black_prefixs : [""]              #可设置多个 比如设置 查询 则查询开头的信息均被拦截 防止审核失败
  alias : ["",""]                   #两两成对,指令替换,"a","b","c","d"代表将a开头替换为b开头,c开头替换为d开头.

  visual_prefixs :                  #虚拟前缀 与white_prefixs配合使用 处理流程自动忽略该前缀 remove_prefix remove_at 需为true时生效
  - prefix: ""                      #虚拟前缀开头 例 你有3个指令 帮助 测试 查询 将 prefix 设置为 工具类 后 则可通过 工具类 帮助 触发机器人
    whiteList: [""]                 #开关状态取决于 white_prefix_mode 为每一个二级指令头设计独立的白名单
    No_White_Response : ""                                   
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" 
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" 

  #开发增强类

  develop_access_token_dir : ""     #开发者测试环境access_token自定义获取地址 默认留空 请留空忽略
  develop_bot_id : "1234"           #开发者环境需自行获取botid 填入 用户请不要设置这两行...开发者调试用
  sandbox_mode : false              #默认false 如果你只希望沙箱频道使用,请改为true
  dev_message_id : false            #在沙盒和测试环境使用无限制msg_id 仅沙盒有效,正式环境请关闭,内测结束后,tx侧未来会移除
  send_error : true                 #将报错用文本发出,避免机器人被审核报无响应
  url_pic_transfer : false          #把图片url(任意来源图链)变成你备案的白名单url 需要较高上下行+ssl+自备案域名+设置白名单域名(暂时不需要)
  idmap_pro : false                 #需开启hash_id配合,高级id转换增强,可以多个真实值bind到同一个虚拟值,对于每个用户,每个群\私聊\判断私聊\频道,都会产生新的虚拟值,但可以多次bind,bind到同一个数字.数据库负担会变大.
  send_delay : 300                  #单位 毫秒 默认300ms 可以视情况减少到100或者50

  title : "Gensokyo © 2023 - Hoshinonyaruko"              #程序的标题 如果多个机器人 可根据标题区分
  custom_bot_name : "Gensokyo全域机器人"                   #自定义机器人名字,会在api调用中返回,默认Gensokyo全域机器人
 
  twoway_echo : false               #是否采用双向echo,根据机器人选择,獭獭\早苗 true 红色问答\椛椛 或者其他 请使用 false
  lazy_message_id : false           #false=message_id 条条准确对应 true=message_id 按时间范围随机对应(适合主动推送bot)前提,有足够多的活跃信息刷新id池
  
  visible_ip : false                #转换url时,如果server_dir是ip true将以ip形式发出url 默认隐藏url 将server_dir配置为自己域名可以转换url
  forward_msg_limit : 3             #发送折叠转发信息时的最大限制条数 若要发转发信息 请设置lazy_message_id为true
  
  #bind指令类 

  bind_prefix : "/bind"             #需设置   #增强配置项  master_id 可触发
  me_prefix : "/me"                 #需设置   #增强配置项  master_id 可触发
  unlock_prefix : "/unlock"         #频道私信卡住了? gsk可以帮到你 在任意子频道发送unlock 你会收到来自机器人的频道私信

  #穿透\cos\oss类配置(可选!)
  frp_port : "0"                    #不使用请保持为0,frp的端口,frp有内外端口,请在frp软件设置gensokyo的port,并将frp显示的对外端口填入这里

  #HTTP API配置

  #正向http
  http_address: ""                  #http监听地址 与websocket独立 示例:0.0.0.0:5700 为空代表不开启
  http_version : 11                 #暂时只支持11
  http_timeout: 5                   #反向 HTTP 超时时间, 单位秒，<5 时将被忽略

  #反向http
  post_url: [""]                    #反向HTTP POST地址列表 为空代表不开启 示例:http://192.168.0.100:5789
  post_secret: [""]                 #密钥
  post_max_retries: [3]             #最大重试,0 时禁用
  post_retries_interval: [1500]     #重试时间,单位毫秒,0 时立即

  #腾讯云配置
  t_COS_BUCKETNAME : ""             #存储桶名称
  t_COS_REGION : ""                 #COS_REGION 所属地域()内的复制进来 可以在控制台查看 https://console.cloud.tencent.com/cos5/bucket, 关于地域的详情见 https://cloud.tencent.com/document/product/436/6224
  t_COS_SECRETID : ""               #用户的 SecretId,建议使用子账号密钥,授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
  t_COS_SECRETKEY : ""              #用户的 SECRETKEY 请腾讯云搜索 api密钥管理 生成并填写.妥善保存 避免泄露
  t_audit : false                   #是否审核内容 请先到控制台开启

  #百度云配置
  b_BOS_BUCKETNAME : ""             #百度智能云-BOS控制台-Bucket列表-需要选择的存储桶-域名发布信息-完整官方域名-填入 形如 hellow.gz.bcebos.com
  b_BCE_AK : ""                     #百度 BCE的 AK 获取方法 https://cloud.baidu.com/doc/BOS/s/Tjwvyrw7a 
  b_BCE_SK : ""                     #百度 BCE的 SK 
  b_audit : 0                       #0 不审核 仅使用oss, 1 使用oss+审核, 2 不使用oss 仅审核

  #阿里云配置
  a_OSS_EndPoint : ""               #形如 https://oss-cn-hangzhou.aliyuncs.com 这里获取 https://oss.console.aliyun.com/bucket/oss-cn-shenzhen/sanaee/overview
  a_OSS_BucketName : ""             #要使用的桶名称,上方EndPoint不包含这个名称,如果有,请填在这里
  a_OSS_AccessKeyId : ""            #阿里云控制台-最右上角点击自己头像-AccessKey管理-然后管理和生成
  a_OSS_AccessKeySecret : ""
  a_audit : false                   #是否审核图片 请先开通阿里云内容安全需企业认证。具体操作 请参见https://help.aliyun.com/document_detail/69806.html

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
