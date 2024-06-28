package template

const ConfigTemplate = `
version: 1
settings:
  #反向ws设置
  ws_address: ["ws://<YOUR_WS_ADDRESS>:<YOUR_WS_PORT>"] # WebSocket服务的地址 支持多个["","",""]
  ws_token: ["","",""]              #连接wss地址时服务器所需的token,按顺序一一对应,如果是ws地址,没有密钥,请留空.
  reconnect_times : 100             #反向ws连接失败后的重试次数,希望一直重试,可设置9999
  heart_beat_interval : 5          #反向ws心跳间隔 单位秒 推荐5-10
  launch_reconnect_times : 1        #启动时尝试反向ws连接次数,建议先打开应用端再开启gensokyo,因为启动时连接会阻塞webui启动,默认只连接一次,可自行增大

  #基础设置
  app_id: 12345                                      # 你的应用ID
  uin : 0                                            # 你的机器人QQ号,点击机器人资料卡查看  
  use_uin : false                                    # false=使用appid作为机器人id,true=使用机器人QQ号,需设置正确的uin
  token: "<YOUR_APP_TOKEN>"                          # 你的应用令牌
  client_secret: "<YOUR_CLIENT_SECRET>"              # 你的客户端密钥
  shard_count: 1                    #分片数量 默认1
  shard_id: 0                       #当前分片id 默认从0开始,详细请看 https://bot.q.qq.com/wiki/develop/api/gateway/reference.html

  #事件订阅
  text_intent:                                       # 请根据公域 私域来选择intent,错误的intent将连接失败
    - "ATMessageEventHandler"                        # 频道at信息
    - "DirectMessageHandler"                         # 私域频道私信(dms)
    # - "ReadyHandler"                               # 连接成功
    # - "ErrorNotifyHandler"                         # 连接关闭
    # - "GuildEventHandler"                          # 频道事件
    # - "MemberEventHandler"                         # 频道成员新增
    # - "ChannelEventHandler"                        # 频道事件
    # - "CreateMessageHandler"                       # 频道不at信息 私域机器人需要开启 公域机器人开启会连接失败
    # - "InteractionHandler"                         # 添加频道互动回应 卡片按钮data回调事件
    # - "GroupATMessageEventHandler"                 # 群at信息 仅频道机器人时候需要注释
    # - "C2CMessageEventHandler"                     # 群私聊 仅频道机器人时候需要注释
    # - "ThreadEventHandler"                         # 频道发帖事件 仅频道私域机器人可用

  #转换类
  global_channel_to_group: true                      # 是否将频道转换成群 默认true
  global_private_to_channel: false                   # 是否将私聊转换成频道 如果是群场景 会将私聊转为群(方便提审\测试)
  global_forum_to_channel: false                     # 是否将频道帖子信息转化为频道 子频道信息 如果开启global_channel_to_group会进一步转换为群信息
  global_interaction_to_message : false              # 是否将按钮和表态回调转化为消息 仅在设置了按钮回调中的message时有效
  global_group_msg_rre_to_message : false            # 是否将用户开关机器人资料页的机器人推送开关 产生的事件转换为文本信息并发送给应用端.false将使用onebotv11的notice类型上报.
  global_group_msg_reject_message : "机器人主动消息被关闭"  # 当开启 global_group_msg_rre_to_message 时,机器人主动信息被关闭将上报的信息. 自行添加intent - GroupMsgRejectHandler
  global_group_msg_receive_message : "机器人主动消息被开启" # 建议设置为无规则复杂随机内容,避免用户指令内容碰撞. 自行添加 intent - GroupMsgReceiveHandler
  hash_id : false                                    # 使用hash来进行idmaps转换,可以让user_id不是123开始的递增值
  idmap_pro : false                                  # 需开启hash_id配合,高级id转换增强,可以多个真实值bind到同一个虚拟值,对于每个用户,每个群\私聊\判断私聊\频道,都会产生新的虚拟值,但可以多次bind,bind到同一个数字.数据库负担会变大.

  #Gensokyo互联类
  server_dir: "<YOUR_SERVER_DIR>"                    # Lotus地址.不带http头的域名或ip,提供图片上传服务的服务器(图床)需要带端口号. 如果需要发base64图,需为公网ip,且开放对应端口
  port: "15630"                                      # Lotus端口.idmaps和图床对外开放的端口号,若要连接到另一个gensokyo,也是链接端口
  backup_port : "5200"                               # 当totus为ture时,port值不再是本地webui的端口,使用lotus_Port来访问webui
  lotus: false                                       # lotus特性默认为false,当为true时,将会连接到另一个lotus为false的gensokyo。使用它提供的图床和idmaps服务(场景:同一个机器人在不同服务器运行,或内网需要发送base64图)。如果需要发送base64图片,需要设置正确的公网server_dir和开放对应的port, lotus鉴权 设置后,从gsk需要保持相同密码来访问主gsk
  lotus_password : "" 
  lotus_without_idmaps: false       #lotus只通过url,图片上传,语音,不通过id转换,在本地当前gsk维护idmaps转换.

  #增强配置项                                           
  master_id : ["1","2"]             #群场景尚未开放获取管理员和列表能力,手动从日志中获取需要设置为管理,的user_id并填入(适用插件有权限判断场景)
  record_sampleRate : 24000         #语音文件的采样率 最高48000 默认24000 单位Khz
  record_bitRate : 24000            #语音文件的比特率 默认25000 代表 25 kbps 最高无限 请根据带宽 您发送的实际码率调整
  card_nick : ""                    #默认为空,连接mirai-overflow时,请设置为非空,这里是机器人对用户称谓,为空为插件获取,mirai不支持
  auto_bind : true                  #测试功能,后期会移除

  #发图相关
  oss_type : 0                      #请完善后方具体配置 完成#腾讯云配置...,0代表配置server dir port服务器自行上传(省钱),1,腾讯cos存储桶 2,百度oss存储桶 3,阿里oss存储桶
  image_sizelimit : 0               #代表kb 腾讯api要求图片1500ms完成传输 如果图片发不出 请提升上行或设置此值 默认为0 不压缩
  image_limit : 100                 #每分钟上传的最大图片数量,可自行增加
  guild_url_image_to_base64 : false #解决频道发不了某些url图片,报错40003问题
  url_pic_transfer : false          #把图片url(任意来源图链)变成你备案的白名单url 需要较高上下行+ssl+自备案域名+设置白名单域名(暂时不需要)
  uploadpicv2_b64: true             #uploadpicv2接口使用base64直接上传 https://www.yuque.com/km57bt/hlhnxg/ig2nk88fccykn6dm
  global_server_temp_qqguild : false                     #需设置server_temp_qqguild,公域私域均可用,以频道为底层发图,速度快,该接口为进阶接口,使用有一定难度.
  server_temp_qqguild : "0"            #在v3图片接口采用固定的子频道号,可以是帖子子频道 https://www.yuque.com/km57bt/hlhnxg/uqmnsno3vx1ytp2q
  server_temp_qqguild_pool : []      #填写v3发图接口的endpoint http://127.0.0.1:12345/uploadpicv3 当填写多个时采用循环方式负载均衡,注,不包括自身,如需要自身也要填写

  #正向ws设置
  ws_server_path : "ws"             #默认监听0.0.0.0:port/ws_server_path 若有安全需求,可不放通port到公网,或设置ws_server_token 若想监听/ 可改为"",若想监听到不带/地址请写nil
  enable_ws_server: true            #是否启用正向ws服务器 监听server_dir:port/ws_server_path
  ws_server_token : "12345"         #正向ws的token 不启动正向ws可忽略 可为空

  #SSL配置类 和 白名单域名自动验证
  identify_file : true               #自动生成域名校验文件,在q.qq.com配置信息URL,在server_dir填入自己已备案域名,正确解析到机器人所在服务器ip地址,机器人即可发送链接
  identify_appids : []               #默认不需要设置,完成SSL配置类+server_dir设置为域名+完成备案+ssl全套设置后,若有多个机器人需要过域名校验(自己名下)可设置,格式为,整数appid,组成的数组
  crt : ""                           #证书路径 从你的域名服务商或云服务商申请签发SSL证书(qq要求SSL) 
  key : ""                           #密钥路径 Apache（crt文件、key文件）示例: "C:\\123.key" \需要双写成\\
  
  #日志类
  developer_log : false             #开启开发者日志 默认关闭
  log_level : 1                     # 0=debug 1=info 2=warning 3=error 默认1
  save_logs : false                 #自动储存日志

  #webui设置
  disable_webui: false              #禁用webui
  server_user_name : "useradmin"    #默认网页面板用户名
  server_user_password : "admin"    #默认网页面板密码

  #指令魔法类
  remove_prefix : false             #是否忽略公域机器人指令前第一个/
  remove_at : false                 #是否忽略公域机器人指令前第一个[CQ:aq,qq=机器人] 场景(公域机器人,但插件未适配at开头)
  remove_bot_at_group : true        #因为群聊机器人不支持发at,开启本开关会自动隐藏群机器人发出的at(不影响频道场景)
  add_at_group : false              #自动在群聊指令前加上at,某些机器人写法特别,必须有at才反应时,请打开,默认请关闭(如果需要at,不需要at指令混杂,请优化代码适配群场景,群场景目前没有at概念)

  white_prefix_mode : false         #公域 过审用 指令白名单模式开关 如果审核严格 请开启并设置白名单指令 以白名单开头的指令会被通过,反之被拦截
  v_white_prefix_mode : true        #虚拟二级指令(虚拟前缀)的白名单,默认开启,有二级指令帮助的效果
  white_prefixs : [""]              #可设置多个 比如设置 机器人 测试 则只有信息以机器人 测试开头会相应 remove_prefix remove_at 需为true时生效
  white_bypass : []                 #格式[1,2,3],白名单不生效的群或用户(私聊时),用于设置自己的灰度沙箱群/灰度沙箱私聊,避免开发测试时反复开关白名单的不便,请勿用于生产环境.
  white_enable : [true,true,true,true,true] #指令白名单生效范围,5个分别对应,频道公(ATMessageEventHandler),频道私(CreateMessageHandler),频道私聊,群,群私聊,改成false,这个范围就不生效指令白名单(使用场景:群全量,频道私域的机器人,或有私信资质的机器人)
  white_bypass_reverse : false      #反转white_bypass的效果,可仅在white_bypass应用white_prefix_mode,场景:您的不同用户群,可以开放不同层次功能,便于您的运营和规化(测试/正式环境)
  No_White_Response : ""            #默认不兜底,强烈建议设置一个友善的兜底回复,告知审核机器人已无隐藏指令,如:你输入的指令不对哦,@机器人来获取可用指令

  black_prefix_mode : false         #公私域 过审用 指令黑名单模式开关 过滤被审核打回的指令不响应 无需改机器人后端
  black_prefixs : [""]              #可设置多个 比如设置 查询 则查询开头的信息均被拦截 防止审核失败
  alias : ["",""]                   #两两成对,指令替换,"a","b","c","d"代表将a开头替换为b开头,c开头替换为d开头.
  enters : ["",""]                  #自动md卡片点击直接触发,小众功能,满足以下条件:应用端支持双向echo+设置了visual_prefixs和whiteList
  enters_except : ["",""]           #自动md卡片点击直接触发,例外,对子按钮生效.
  auto_withdraw : []                #仅当应用端实现了双向echo可用.实现不难,可以去找对应开发者去提需求.
  auto_withdraw_time : 30           #30秒
  visual_prefixs_bypass : []        #要绕过二级指令白名单的指令,比如需要用户回答自定义内容的指令.

  visual_prefixs :                  #虚拟前缀 与white_prefixs配合使用 处理流程自动忽略该前缀 remove_prefix remove_at 需为true时生效
  - prefix: ""                      #虚拟前缀开头 例 你有3个指令 帮助 测试 查询 将 prefix 设置为 工具类 后 则可通过 工具类 帮助 触发机器人
    whiteList: [""]                 #开关状态取决于 white_prefix_mode 为每一个二级指令头设计独立的白名单
    No_White_Response : ""          #带有*代表不忽略掉,但是应用二级白名单的普通指令   如果  whiteList=邀请机器人 就是邀请机器人按钮                    
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
  save_error : false                #将保存保存在log文件夹,方便开发者定位发送错误.
  downtime_message : "我正在维护中~请不要担心,维护结束就回来~维护时间:(1小时)"

  #增长营销类(推荐gensokyo-broadcast项目)
  self_introduce : ["",""]          #自我介绍,可设置多个随机发送,当不为空时,机器人被邀入群会发送自定义自我介绍 需手动添加新textintent   - "GroupAddRobotEventHandler"   - "GroupDelRobotEventHandler"


  #API修改
  get_g_list_all_guilds : false     #在获取群列表api时,轮询获取全部的频道列表(api一次只能获取100个),建议仅在广播公告通知等特别场景时开启.
  get_g_list_delay : 500            #轮询时的延迟时间,毫秒数.
  get_g_list_guilds_type : 0        #0=全部返回,1=获取第1个子频道.以此类推.可以缩减返回值的大小.
  get_g_list_guilds : "10"          #在获取群列表api时,一次返回的频道数量.这里是string,不要去掉引号.最大100(5分钟内连续请求=翻页),获取全部请开启get_g_list_return_guilds.
  get_g_list_return_guilds : true   #获取群列表时是否返回频道列表.
  forward_msg_limit : 3             #发送折叠转发信息时的最大限制条数 若要发转发信息 请设置lazy_message_id为true
  custom_bot_name : "Gensokyo全域机器人"   #自定义api返回的机器人名字,会在api调用中返回,默认Gensokyo全域机器人
  transform_api_ids : true          #对get_group_menmber_list\get_group_member_info\get_group_list生效,是否在其中返回转换后的值(默认转换,不转换请自行处理插件逻辑,比如调用gsk的http api转换)
  auto_put_interaction : false      #自动回应按钮回调的/interactions/{interaction_id} 注本api需要邮件申请,详细方法参考群公告:196173384
  put_interaction_delay : 0         #单位毫秒 表示回应已收到回调类型的按钮的毫秒数 会按用户进行区分 非全局delay

  #Onebot修改
  twoway_echo : false               #是否采用双向echo,根据机器人选择,獭獭\早苗 true 红色问答\椛椛 或者其他 请使用 false
  array: false                                       # 连接trss云崽请开启array
  native_ob11 : false               #如果你的机器人收到事件报错,请开启此选项增加兼容性

  #URL相关
  visible_ip : false                #转换url时,如果server_dir是ip true将以ip形式发出url 默认隐藏url 将server_dir配置为自己域名可以转换url
  url_to_qrimage : false            #将信息中的url转换为二维码单独作为图片发出,需要同时设置  #SSL配置类 机器人发送URL设置 的 transfer_url 为 true visible_ip也需要为true
  qr_size : 200                     #二维码尺寸,单位像素
  transfer_url : true                #默认开启,关闭后自理url发送,配置server_dir为你的域名,配置crt和key后,将域名/url和/image在q.qq.com后台通过校验,自动使用302跳转处理机器人发出的所有域名.

  #框架修改
  title : "Gensokyo © 2023 - Hoshinonyaruko"              #程序的标题 如果多个机器人 可根据标题区分
  frp_port : "0"                    #不使用请保持为0,frp的端口,frp有内外端口,请在frp软件设置gensokyo的port,并将frp显示的对外端口填入这里
 
  #MD相关
  custom_template_id : ""           #自动转换图文信息到md所需要的id *需要应用端支持双方向echo
  keyboard_id : ""                  #自动转换图文信息到md所需要的按钮id *需要应用端支持双方向echo
  native_md : false                 #自动转换图文信息到md,使用原生markdown能力.
  enters_as_block : false           #自动转换图文信息到md,\r \r\n \n 替换为空格.

  #发送行为修改
  lazy_message_id : false           #false=message_id 条条准确对应 true=message_id 按时间范围随机对应(适合主动推送bot)前提,有足够多的活跃信息刷新id池
  ramdom_seq : false                #当多开gensokyo时,如果遇到群信息只能发出一条,请开启每个gsk的此项.(建议使用一个gsk连接多个应用)
  bot_forum_title : "机器人帖子"                      # 机器人发帖子回复默认标题 
  AMsgRetryAsPMsg_Count : 30        #当主动信息发送失败时,自动转为后续的被动信息发送,需要开启Lazy message id,该配置项为所有群、频道的主动转被动消息队列最大长度,建议30-100,无上限
  send_delay : 300                  #单位 毫秒 默认300ms 可以视情况减少到100或者50
  enableChangeWord : false          #敏感词替换系统,具有IN和OUT两个文本维度,会在运行目录下释放txt文件,一行一个,格式为aaa####bbb,作用是将aaa替换为bbb,输入替换是对用户输入进行替换,输出则是替换机器人发出的文本信息.
  defaultChangeWord : "*"           #默认替换词,当开启

  #错误临时修复类
  fix_11300: false                  #修复11300报错,需要在develop_bot_id填入自己机器人的appid. 11300原因暂时未知,临时修复方案.
  http_only_bot : false             #这个配置项会自动配置,请不要修改,保持false.
  
  #内置指令类
  bind_prefix : "/bind"             #需设置   #增强配置项  master_id 可触发
  me_prefix : "/me"                 #需设置   #增强配置项  master_id 可触发
  unlock_prefix : "/unlock"         #频道私信卡住了? gsk可以帮到你 在任意子频道发送unlock 你会收到来自机器人的频道私信
  link_prefix : "/link"             #友情链接配置 配置custom_template_id后可用(https://www.yuque.com/km57bt/hlhnxg/tzbr84y59dbz6pib)
  auto_link : false                 #友情链接最高礼仪,机器人被添加到群内时发送友情链接.
  music_prefix : "点歌"             #[CQ:music,type=qq,id=123] 在消息文本组合qq音乐歌曲id,可以发送点歌,这是歌曲按钮第二个按钮的填充内容,应为你的机器人点歌插件的指令.
  link_bots : ["",""]               #发送友情链接时 下方按钮携带的机器人 格式 "appid-qq-name","appid-qq-name"或"http://xxx.com-文字" 链接中的-号自行用%2D替换 如 cgi-bin替换为cgi%2Dbin
  link_text : ""                    #友情链接文本 不可为空!
  link_pic : ""                     #友情链接图片 可为空 需url图片 可带端口 不填可能会有显示错误
  link_lines : 2                    #内置的/link指令按钮列数(默认一行2个按钮)
  link_num : 6                      #在link_bots随机选n个bot上友情链接,在link_bots中随机n个,避免用户选择困难,少即是多

  #HTTP API配置-正向http
  http_address: ""                  #http监听地址 与websocket独立 示例:0.0.0.0:5700 为空代表不开启
  http_access_token: ""             #http访问令牌
  http_version : 11                 #暂时只支持11
  http_timeout: 5                   #反向 HTTP 超时时间, 单位秒，<5 时将被忽略

  #HTTP API配置-反向http
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
