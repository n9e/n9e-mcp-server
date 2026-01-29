package types

// N9eResponse represents Nightingale unified response format
type N9eResponse[T any] struct {
	Dat T      `json:"dat"`
	Err string `json:"err"`
}

// PageResp represents paginated response
type PageResp[T any] struct {
	List  []T   `json:"list"`
	Total int64 `json:"total"`
}

// IdName represents ID and name pair
type IdName struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

// AlertCurEvent represents active alert event
type AlertCurEvent struct {
	// Basic identification
	Id           int64  `json:"id"`
	Hash         string `json:"hash"`
	Cate         string `json:"cate"`
	Cluster      string `json:"cluster"`
	DatasourceId int64  `json:"datasource_id"`

	// Rule information
	RuleId       int64  `json:"rule_id"`
	RuleName     string `json:"rule_name"`
	RuleNote     string `json:"rule_note"`
	RuleProd     string `json:"rule_prod"`
	RuleAlgo     string `json:"rule_algo"`
	Severity     int    `json:"severity"`

	// Query related
	PromQl           string `json:"prom_ql"`
	PromForDuration  int    `json:"prom_for_duration"`
	PromEvalInterval int    `json:"prom_eval_interval"`

	// Notification configuration
	Callbacks       []string `json:"callbacks"`
	RunbookUrl      string   `json:"runbook_url"`
	NotifyRecovered int      `json:"notify_recovered"`
	NotifyChannels  []string `json:"notify_channels"`
	NotifyGroups    []string `json:"notify_groups"`
	NotifyGroupsObj []IdName `json:"notify_groups_obj"`

	// Target information
	TargetIdent string `json:"target_ident"`
	TargetNote  string `json:"target_note"`

	// Trigger information
	TriggerTime      int64             `json:"trigger_time"`
	TriggerValue     string            `json:"trigger_value"`
	TriggerValues    string            `json:"trigger_values"`
	FirstTriggerTime int64             `json:"first_trigger_time"`

	// Tags and annotations
	Tags         []string          `json:"tags"`
	TagsMap      map[string]string `json:"tags_map"`
	OriginalTags []string          `json:"original_tags"`
	Annotations  map[string]string `json:"annotations"`

	// Business group
	GroupId   int64  `json:"group_id"`
	GroupName string `json:"group_name"`

	// Status
	Status          int    `json:"status"`
	Claimant        string `json:"claimant"`
	NotifyCurNumber int    `json:"notify_cur_number"`

	// Extended information
	ExtraConfig  map[string]any   `json:"extra_config,omitempty"`
	ExtraInfo    []string         `json:"extra_info,omitempty"`
	ExtraInfoMap []map[string]any `json:"extra_info_map,omitempty"`
}

// AlertHisEvent represents historical alert event
type AlertHisEvent struct {
	AlertCurEvent
	IsRecovered int   `json:"is_recovered"`
	RecoverTime int64 `json:"recover_time"`
}

// AlertRule represents alert rule
type AlertRule struct {
	Id                   int64             `json:"id"`
	GroupId              int64             `json:"group_id"`
	Cate                 string            `json:"cate"`
	DatasourceIds        any               `json:"datasource_ids,omitempty"`
	DatasourceQueries    any               `json:"datasource_queries,omitempty"`
	Cluster              string            `json:"cluster"`
	Name                 string            `json:"name"`
	Note                 string            `json:"note"`
	Prod                 string            `json:"prod"`
	Algorithm            string            `json:"algorithm"`
	AlgoParams           any               `json:"algo_params,omitempty"`
	Delay                int               `json:"delay"`
	Severity             int               `json:"severity"`
	Severities           any               `json:"severities,omitempty"`
	Disabled             int               `json:"disabled"`
	PromForDuration      int               `json:"prom_for_duration"`
	PromQl               string            `json:"prom_ql"`
	RuleConfig           any               `json:"rule_config,omitempty"`
	PromEvalInterval     int               `json:"prom_eval_interval"`
	EnableStime          any               `json:"enable_stime,omitempty"`
	EnableStimes         any               `json:"enable_stimes,omitempty"`
	EnableEtime          any               `json:"enable_etime,omitempty"`
	EnableEtimes         any               `json:"enable_etimes,omitempty"`
	EnableDaysOfWeek     any               `json:"enable_days_of_week,omitempty"`
	EnableDaysOfWeeks    any               `json:"enable_days_of_weeks,omitempty"`
	EnableInBG           int               `json:"enable_in_bg"`
	NotifyRecovered      int               `json:"notify_recovered"`
	NotifyChannels       any               `json:"notify_channels,omitempty"`
	NotifyGroups         any               `json:"notify_groups,omitempty"`
	NotifyGroupsObj      any               `json:"notify_groups_obj,omitempty"`
	NotifyRepeatStep     int               `json:"notify_repeat_step"`
	NotifyMaxNumber      int               `json:"notify_max_number"`
	NotifyVersion        int               `json:"notify_version"`
	NotifyRuleIds        any               `json:"notify_rule_ids,omitempty"`
	RecoverDuration      int64             `json:"recover_duration"`
	Callbacks            any               `json:"callbacks,omitempty"`
	RunbookUrl           string            `json:"runbook_url"`
	AppendTags           any               `json:"append_tags,omitempty"`
	Annotations          any               `json:"annotations,omitempty"`
	ExtraConfig          any               `json:"extra_config,omitempty"`
	CreateAt             int64             `json:"create_at"`
	CreateBy             string            `json:"create_by"`
	UpdateAt             int64             `json:"update_at"`
	UpdateBy             string            `json:"update_by"`
}

// NotifyRule represents notification rule
type NotifyRule struct {
	Id              int64            `json:"id"`
	Name            string           `json:"name"`
	Description     string           `json:"description"`
	Enable          bool             `json:"enable"`
	UserGroupIds    []int64          `json:"user_group_ids"`
	PipelineConfigs []PipelineConfig `json:"pipeline_configs"`
	NotifyConfigs   []NotifyConfig   `json:"notify_configs"`
	ExtraConfig     any              `json:"extra_config,omitempty"`
	CreateAt        int64            `json:"create_at"`
	CreateBy        string           `json:"create_by"`
	UpdateAt        int64            `json:"update_at"`
	UpdateBy        string           `json:"update_by"`
}

// PipelineConfig represents event processing pipeline configuration
type PipelineConfig struct {
	PipelineId int64 `json:"pipeline_id"`
	Enable     bool  `json:"enable"`
}

// NotifyConfig represents notification configuration
type NotifyConfig struct {
	ChannelID  int64          `json:"channel_id"`
	TemplateID int64          `json:"template_id"`
	Params     map[string]any `json:"params"`
	Type       string         `json:"type"`
	Severities []int          `json:"severities"`
	TimeRanges []TimeRange    `json:"time_ranges"`
	LabelKeys  []TagFilter    `json:"label_keys"`
	Attributes []TagFilter    `json:"attributes"`
}

// TimeRange represents time range
type TimeRange struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Weekdays  []int  `json:"weekdays"`
}

// TagFilter represents tag filter
type TagFilter struct {
	Key   string `json:"key"`
	Func  string `json:"func"`
	Value string `json:"value"`
}

// AlertSubscribe represents alert subscription
type AlertSubscribe struct {
	Id               int64  `json:"id"`
	Name             string `json:"name"`
	Disabled         int    `json:"disabled"`
	GroupId          int64  `json:"group_id"`
	Prod             string `json:"prod"`
	Cate             string `json:"cate"`
	DatasourceIds    any    `json:"datasource_ids,omitempty"`
	Cluster          string `json:"cluster"`
	RuleId           int64  `json:"rule_id"`
	RuleIds          any    `json:"rule_ids,omitempty"`
	RuleName         string `json:"rule_name,omitempty"`
	RuleNames        any    `json:"rule_names,omitempty"`
	Severities       any    `json:"severities,omitempty"`
	ForDuration      int64  `json:"for_duration"`
	Tags             any    `json:"tags,omitempty"`
	RedefineSeverity int    `json:"redefine_severity"`
	NewSeverity      int    `json:"new_severity"`
	RedefineChannels int    `json:"redefine_channels"`
	NewChannels      string `json:"new_channels"`
	UserGroupIds     string `json:"user_group_ids"`
	UserGroups       any    `json:"user_groups,omitempty"`
	RedefineWebhooks int    `json:"redefine_webhooks"`
	Webhooks         any    `json:"webhooks,omitempty"`
	ExtraConfig      any    `json:"extra_config,omitempty"`
	Note             string `json:"note"`
	BusiGroups       any    `json:"busi_groups,omitempty"`
	NotifyRuleIds    any    `json:"notify_rule_ids,omitempty"`
	NotifyVersion    int    `json:"notify_version"`
	CreateAt         int64  `json:"create_at"`
	CreateBy         string `json:"create_by"`
	UpdateAt         int64  `json:"update_at"`
	UpdateBy         string `json:"update_by"`
}

// Target represents monitored object
type Target struct {
	Id           int64             `json:"id"`
	GroupId      int64             `json:"group_id"`
	GroupObj     *BusiGroup        `json:"group_obj,omitempty"`
	Ident        string            `json:"ident"`
	Note         string            `json:"note"`
	Tags         []string          `json:"tags"`
	TagsMap      map[string]string `json:"tags_map"`
	HostIp       string            `json:"host_ip"`
	AgentVersion string            `json:"agent_version"`
	TargetUp     int               `json:"target_up"`
	EngineName   string            `json:"engine_name"`
	UnixTime     int64             `json:"unix_time"`
	UpdateAt     int64             `json:"update_at"`
	Offset       int64             `json:"offset"`
	OS           string            `json:"os"`
	Arch         string            `json:"arch"`
	RemoteAddr   string            `json:"remote_addr"`
	CpuNum       int               `json:"cpu_num"`
	MemSize      int64             `json:"mem_size"`
}

// Datasource represents data source
type Datasource struct {
	Id             int64          `json:"id"`
	Name           string         `json:"name"`
	Identifier     string         `json:"identifier"`
	Description    string         `json:"description"`
	PluginId       int64          `json:"plugin_id"`
	PluginType     string         `json:"plugin_type"`
	PluginTypeName string         `json:"plugin_type_name"`
	Category       string         `json:"category"`
	ClusterName    string         `json:"cluster_name"`
	Settings       map[string]any `json:"settings"`
	Status         string         `json:"status"`
	HTTP           DatasourceHTTP `json:"http"`
	Auth           DatasourceAuth `json:"auth"`
	IsDefault      bool           `json:"is_default"`
	CreatedAt      int64          `json:"created_at"`
	CreatedBy      string         `json:"created_by"`
	UpdatedAt      int64          `json:"updated_at"`
	UpdatedBy      string         `json:"updated_by"`
}

// DatasourceHTTP represents datasource HTTP configuration
type DatasourceHTTP struct {
	Timeout             int64             `json:"timeout"`
	DialTimeout         int64             `json:"dial_timeout"`
	MaxIdleConnsPerHost int               `json:"max_idle_conns_per_host"`
	Url                 string            `json:"url"`
	Urls                []string          `json:"urls"`
	Headers             map[string]string `json:"headers"`
	TLS                 DatasourceTLS     `json:"tls"`
}

// DatasourceTLS represents TLS configuration
type DatasourceTLS struct {
	SkipTlsVerify bool   `json:"skip_tls_verify"`
	ServerName    string `json:"server_name"`
	MinVersion    string `json:"min_version"`
	MaxVersion    string `json:"max_version"`
}

// DatasourceAuth represents datasource authentication configuration
type DatasourceAuth struct {
	BasicAuth         bool   `json:"basic_auth"`
	BasicAuthUser     string `json:"basic_auth_user"`
	BasicAuthPassword string `json:"basic_auth_password"`
}

// AlertMute represents alert mute/silence
type AlertMute struct {
	Id            int64  `json:"id"`
	GroupId       int64  `json:"group_id"`
	Note          string `json:"note"`
	Cate          string `json:"cate"`
	Prod          string `json:"prod"`
	DatasourceIds any    `json:"datasource_ids,omitempty"`
	Cluster       string `json:"cluster"`
	Tags          any    `json:"tags,omitempty"`
	Cause         string `json:"cause"`
	Btime         int64  `json:"btime"`
	Etime         int64  `json:"etime"`
	Severities    any    `json:"severities,omitempty"`
	Disabled      int    `json:"disabled"`
	Activated     int    `json:"activated,omitempty"`
	MuteTimeType  int    `json:"mute_time_type"`
	PeriodicMutes any    `json:"periodic_mutes,omitempty"`
	CreateAt      int64  `json:"create_at"`
	CreateBy      string `json:"create_by"`
	UpdateAt      int64  `json:"update_at"`
	UpdateBy      string `json:"update_by"`
}

// BusiGroup represents business group
type BusiGroup struct {
	Id          int64  `json:"id"`
	Name        string `json:"name"`
	LabelEnable int    `json:"label_enable"`
	LabelValue  string `json:"label_value"`
	CreateAt    int64  `json:"create_at"`
	CreateBy    string `json:"create_by"`
	UpdateAt    int64  `json:"update_at"`
	UpdateBy    string `json:"update_by"`
}

// EventPipeline represents event processing pipeline
type EventPipeline struct {
	Id               int64             `json:"id"`
	Name             string            `json:"name"`
	Typ              string            `json:"typ"`
	UseCase          string            `json:"use_case"`
	TriggerMode      string            `json:"trigger_mode"`
	Disabled         bool              `json:"disabled"`
	TeamIds          []int64           `json:"team_ids"`
	TeamNames        []string          `json:"team_names,omitempty"`
	Description      string            `json:"description"`
	FilterEnable     bool              `json:"filter_enable"`
	LabelFilters     []TagFilter       `json:"label_filters"`
	AttrFilters      []TagFilter       `json:"attribute_filters"`
	ProcessorConfigs []ProcessorConfig `json:"processors"`
	Nodes            []WorkflowNode    `json:"nodes,omitempty"`
	Connections      any               `json:"connections,omitempty"`
	Inputs           []InputVariable   `json:"inputs,omitempty"`
	CreateAt         int64             `json:"create_at"`
	CreateBy         string            `json:"create_by"`
	UpdateAt         int64             `json:"update_at"`
	UpdateBy         string            `json:"update_by"`
}

// ProcessorConfig represents processor configuration
type ProcessorConfig struct {
	Typ    string `json:"typ"`
	Config any    `json:"config"`
}

// WorkflowNode represents workflow node
type WorkflowNode struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Config   any    `json:"config,omitempty"`
	Position any    `json:"position,omitempty"`
}

// InputVariable represents input variable
type InputVariable struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

// EventPipelineExecution represents event pipeline execution record
type EventPipelineExecution struct {
	Id             string `json:"id"`
	PipelineId     int64  `json:"pipeline_id"`
	PipelineName   string `json:"pipeline_name"`
	EventId        int64  `json:"event_id"`
	Mode           string `json:"mode"`
	Status         string `json:"status"`
	NodeResults    string `json:"node_results"`
	ErrorMessage   string `json:"error_message"`
	ErrorNode      string `json:"error_node"`
	CreatedAt      int64  `json:"created_at"`
	FinishedAt     int64  `json:"finished_at"`
	DurationMs     int64  `json:"duration_ms"`
	TriggerBy      string `json:"trigger_by"`
	InputsSnapshot string `json:"inputs_snapshot,omitempty"`
}

// User represents user
type User struct {
	Id             int64          `json:"id"`
	Username       string         `json:"username"`
	Nickname       string         `json:"nickname"`
	Phone          string         `json:"phone"`
	Email          string         `json:"email"`
	Portrait       string         `json:"portrait"`
	Roles          []string       `json:"roles"`
	Contacts       map[string]any `json:"contacts"`
	Maintainer     int            `json:"maintainer"`
	CreateAt       int64          `json:"create_at"`
	CreateBy       string         `json:"create_by"`
	UpdateAt       int64          `json:"update_at"`
	UpdateBy       string         `json:"update_by"`
	Belong         string         `json:"belong"`
	Admin          bool           `json:"admin"`
	LastActiveTime int64          `json:"last_active_time"`
}

// UserGroup represents user group/team
type UserGroup struct {
	Id         int64       `json:"id"`
	Name       string      `json:"name"`
	Note       string      `json:"note"`
	CreateAt   int64       `json:"create_at"`
	CreateBy   string      `json:"create_by"`
	UpdateAt   int64       `json:"update_at"`
	UpdateBy   string      `json:"update_by"`
	Users      []User      `json:"users,omitempty"`
	BusiGroups []BusiGroup `json:"busi_groups,omitempty"`
}

// UserGroupDetail represents user group details (including members)
type UserGroupDetail struct {
	UserGroup UserGroup `json:"user_group"`
	Users     []User    `json:"users"`
}

// PeriodicMute represents periodic mute rule
type PeriodicMute struct {
	EnableStime      string `json:"enable_stime"`
	EnableEtime      string `json:"enable_etime"`
	EnableDaysOfWeek string `json:"enable_days_of_week"`
}
