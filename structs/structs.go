package structs

type FriendData struct {
	Nickname string `json:"nickname"`
	Remark   string `json:"remark"`
	UserID   string `json:"user_id"`
}

type Settings struct {
	//反向ws设置
	WsAddress           []string `yaml:"ws_address"`
	WsToken             []string `yaml:"ws_token"`
	ReconnecTimes       int      `yaml:"reconnect_times"`
	HeartBeatInterval   int      `yaml:"heart_beat_interval"`
	LaunchReconectTimes int      `yaml:"launch_reconnect_times"`
	//基础配置
	AppID        uint64 `yaml:"app_id"`
	Uin          int64  `yaml:"uin"`
	Token        string `yaml:"token"`
	ClientSecret string `yaml:"client_secret"`
	ShardCount   int    `yaml:"shard_count"`
	ShardID      int    `yaml:"shard_id"`
	UseUin       bool   `yaml:"use_uin"`
	//事件订阅类
	TextIntent []string `yaml:"text_intent"`
	//转换类
	GlobalChannelToGroup                     bool   `yaml:"global_channel_to_group"`
	GlobalPrivateToChannel                   bool   `yaml:"global_private_to_channel"`
	GlobalForumToChannel                     bool   `yaml:"global_forum_to_channel"`
	GlobalInteractionToMessage               bool   `yaml:"global_interaction_to_message"`
	GlobalGroupMsgRejectReciveEventToMessage bool   `yaml:"global_group_msg_rre_to_message"`
	GlobalGroupMsgRejectMessage              string `yaml:"global_group_msg_reject_message"`
	GlobalGroupMsgReceiveMessage             string `yaml:"global_group_msg_receive_message"`
	HashID                                   bool   `yaml:"hash_id"`
	IdmapPro                                 bool   `yaml:"idmap_pro"`
	//gensokyo互联类
	Server_dir            string `yaml:"server_dir"`
	Port                  string `yaml:"port"`
	BackupPort            string `yaml:"backup_port"`
	Lotus                 bool   `yaml:"lotus"`
	LotusPassword         string `yaml:"lotus_password"`
	LotusWithoutIdmaps    bool   `yaml:"lotus_without_idmaps"`
	LotusWithoutUploadPic bool   `yaml:"lotus_without_uploadpic"`
	LotusGrpc             bool   `yaml:"lotus_grpc"`
	LotusGrpcPort         int    `yaml:"lotus_grpc_port"`
	//增强配置
	MasterID         []string `yaml:"master_id"`
	RecordSampleRate int      `yaml:"record_sampleRate"`
	RecordBitRate    int      `yaml:"record_bitRate"`
	CardAndNick      string   `yaml:"card_nick"`
	AutoBind         bool     `yaml:"auto_bind"`
	//发图相关
	OssType                 int      `yaml:"oss_type"`
	ImageLimit              int      `yaml:"image_sizelimit"`
	ImageLimitB             int      `yaml:"image_limit"`
	GuildUrlImageToBase64   bool     `yaml:"guild_url_image_to_base64"`
	UrlPicTransfer          bool     `yaml:"url_pic_transfer"`
	UploadPicV2Base64       bool     `yaml:"uploadpicv2_b64"`
	GlobalServerTempQQguild bool     `yaml:"global_server_temp_qqguild"`
	ServerTempQQguild       string   `yaml:"server_temp_qqguild"`
	ServerTempQQguildPool   []string `yaml:"server_temp_qqguild_pool"`
	//正向ws设置
	WsServerPath   string `yaml:"ws_server_path"`
	EnableWsServer bool   `yaml:"enable_ws_server"`
	WsServerToken  string `yaml:"ws_server_token"`
	//ssl和链接转换类
	IdentifyFile   bool    `yaml:"identify_file"`
	IdentifyAppids []int64 `yaml:"identify_appids"`
	Crt            string  `yaml:"crt"`
	Key            string  `yaml:"key"`
	//日志类
	DeveloperLog bool `yaml:"developer_log"`
	LogLevel     int  `yaml:"log_level"`
	SaveLogs     bool `yaml:"save_logs"`
	//webui相关
	DisableWebui bool   `yaml:"disable_webui"`
	Username     string `yaml:"server_user_name"`
	Password     string `yaml:"server_user_password"`
	//指令魔法类
	RemovePrefix        bool                 `yaml:"remove_prefix"`
	RemoveAt            bool                 `yaml:"remove_at"`
	RemoveBotAtGroup    bool                 `yaml:"remove_bot_at_group"`
	AddAtGroup          bool                 `yaml:"add_at_group"`
	WhitePrefixMode     bool                 `yaml:"white_prefix_mode"`
	VwhitePrefixMode    bool                 `yaml:"v_white_prefix_mode"`
	WhitePrefixs        []string             `yaml:"white_prefixs"`
	WhiteBypass         []int64              `yaml:"white_bypass"`
	WhiteEnable         []bool               `yaml:"white_enable"`
	WhiteBypassRevers   bool                 `yaml:"white_bypass_reverse"`
	NoWhiteResponse     string               `yaml:"No_White_Response"`
	BlackPrefixMode     bool                 `yaml:"black_prefix_mode"`
	BlackPrefixs        []string             `yaml:"black_prefixs"`
	Alias               []string             `yaml:"alias"`
	Enters              []string             `yaml:"enters"`
	EntersExcept        []string             `yaml:"enters_except"`
	VisualPrefixs       []VisualPrefixConfig `yaml:"visual_prefixs"`
	AutoWithdraw        []string             `yaml:"auto_withdraw"`
	AutoWithdrawTime    int                  `yaml:"auto_withdraw_time"`
	VisualPrefixsBypass []string             `yaml:"visual_prefixs_bypass"`
	//开发增强类
	DevlopAcDir     string `yaml:"develop_access_token_dir"`
	DevBotid        string `yaml:"develop_bot_id"`
	SandBoxMode     bool   `yaml:"sandbox_mode"`
	DevMessgeID     bool   `yaml:"dev_message_id"`
	SendError       bool   `yaml:"send_error"`
	SaveError       bool   `yaml:"save_error"`
	DowntimeMessage string `yaml:"downtime_message"`
	MemoryMsgid     bool   `yaml:"memory_msgid"`
	//增长营销类
	SelfIntroduce []string `yaml:"self_introduce"`
	//api修改
	GetGroupListAllGuilds    bool     `yaml:"get_g_list_all_guilds"`
	GetGroupListGuilds       string   `yaml:"get_g_list_guilds"`
	GetGroupListReturnGuilds bool     `yaml:"get_g_list_return_guilds"`
	GetGroupListGuidsType    int      `yaml:"get_g_list_guilds_type"`
	GetGroupListDelay        int      `yaml:"get_g_list_delay"`
	ForwardMsgLimit          int      `yaml:"forward_msg_limit"`
	CustomBotName            string   `yaml:"custom_bot_name"`
	TransFormApiIds          bool     `yaml:"transform_api_ids"`
	AutoPutInteraction       bool     `yaml:"auto_put_interaction"`
	PutInteractionDelay      int      `yaml:"put_interaction_delay"`
	PutInteractionExcept     []string `yaml:"put_interaction_except"`
	//onebot修改
	TwoWayEcho       bool `yaml:"twoway_echo"`
	Array            bool `yaml:"array"`
	NativeOb11       bool `yaml:"native_ob11"`
	DisableErrorChan bool `yaml:"disable_error_chan"`
	//url相关
	VisibleIp    bool `yaml:"visible_ip"`
	UrlToQrimage bool `yaml:"url_to_qrimage"`
	QrSize       int  `yaml:"qr_size"`
	TransferUrl  bool `yaml:"transfer_url"`
	//框架修改
	Title   string `yaml:"title"`
	FrpPort string `yaml:"frp_port"`
	//MD相关
	CustomTemplateID string `yaml:"custom_template_id"`
	KeyBoardID       string `yaml:"keyboard_id"`
	NativeMD         bool   `yaml:"native_md"`
	EntersAsBlock    bool   `yaml:"enters_as_block"`
	//发送行为修改
	LazyMessageId     bool   `yaml:"lazy_message_id"`
	RamDomSeq         bool   `yaml:"ramdom_seq"`
	BotForumTitle     string `yaml:"bot_forum_title"`
	AtoPCount         int    `yaml:"AMsgRetryAsPMsg_Count"`
	SendDelay         int    `yaml:"send_delay"`
	EnableChangeWord  bool   `yaml:"enableChangeWord"`
	DefaultChangeWord string `yaml:"defaultChangeWord"`
	//错误临时修复类
	Fix11300          bool `yaml:"fix_11300"`
	HttpOnlyBot       bool `yaml:"http_only_bot"`
	DoNotReplaceAppid bool `yaml:"do_not_replace_appid"`
	//内置指令
	BindPrefix   string   `yaml:"bind_prefix"`
	MePrefix     string   `yaml:"me_prefix"`
	UnlockPrefix string   `yaml:"unlock_prefix"`
	LinkPrefix   string   `yaml:"link_prefix"`
	AutoLink     bool     `yaml:"auto_link"`
	MusicPrefix  string   `yaml:"music_prefix"`
	LinkBots     []string `yaml:"link_bots"`
	LinkText     string   `yaml:"link_text"`
	LinkPic      string   `yaml:"link_pic"`
	LinkLines    int      `yaml:"link_lines"`
	LinkNum      int      `yaml:"link_num"`
	//HTTP API配置
	HttpAddress         string   `yaml:"http_address"`
	AccessToken         string   `yaml:"http_access_token"`
	HttpVersion         int      `yaml:"http_version"`
	HttpTimeOut         int      `yaml:"http_timeout"`
	PostUrl             []string `yaml:"post_url"`
	PostSecret          []string `yaml:"post_secret"`
	PostMaxRetries      []int    `yaml:"post_max_retries"`
	PostRetriesInterval []int    `yaml:"post_retries_interval"`
	//腾讯云
	TencentBucketName   string `yaml:"t_COS_BUCKETNAME"`
	TencentBucketRegion string `yaml:"t_COS_REGION"`
	TencentCosSecretid  string `yaml:"t_COS_SECRETID"`
	TencentSecretKey    string `yaml:"t_COS_SECRETKEY"`
	TencentAudit        bool   `yaml:"t_audit"`
	//百度云
	BaiduBOSBucketName string `yaml:"b_BOS_BUCKETNAME"`
	BaiduBCEAK         string `yaml:"b_BCE_AK"`
	BaiduBCESK         string `yaml:"b_BCE_SK"`
	BaiduAudit         int    `yaml:"b_audit"`
	//阿里云
	AliyunEndpoint        string `yaml:"a_OSS_EndPoint"`
	AliyunAccessKeyId     string `yaml:"a_OSS_AccessKeyId"`
	AliyunAccessKeySecret string `yaml:"a_OSS_AccessKeySecret"`
	AliyunBucketName      string `yaml:"a_OSS_BucketName"`
	AliyunAudit           bool   `yaml:"a_audit"`
}

type VisualPrefixConfig struct {
	Prefix          string   `yaml:"prefix"`
	WhiteList       []string `yaml:"whiteList"`
	NoWhiteResponse string   `yaml:"No_White_Response"`
}

type InterfaceBody struct {
	Content        string   `json:"content"`
	State          int      `json:"state"`
	PromptKeyboard []string `json:"prompt_keyboard,omitempty"`
	ActionButton   int      `json:"action_button,omitempty"`
	CallbackData   string   `json:"callback_data,omitempty"`
}
