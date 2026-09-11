package model

import (
	"github.com/jumpserver-dev/sdk-go/common"
)

type ConnectToken struct {
	Id       string     `json:"id"`
	User     User       `json:"user"`
	Value    string     `json:"value"`
	Account  Account    `json:"account"`
	Actions  Actions    `json:"actions"`
	Asset    Asset      `json:"asset"`
	Protocol string     `json:"protocol"`
	Domain   *Domain    `json:"domain"`
	Zone     *Domain    `json:"zone"`
	Gateway  *Gateway   `json:"gateway"`
	ExpireAt ExpireInfo `json:"expire_at"`
	OrgId    string     `json:"org_id"`
	OrgName  string     `json:"org_name"`
	Platform Platform   `json:"platform"`

	ConnectMethod ConnectMethod `json:"connect_method"`

	ConnectOptions ConnectOptions `json:"connect_options"`

	CommandFilterACLs []CommandACL `json:"command_filter_acls"`

	ClipboardPolicy  *ClipboardPolicy  `json:"clipboard_policy"`
	DataMaskingRules []DataMaskingRule `json:"data_masking_rules"`

	Ticket           *ObjectId           `json:"from_ticket,omitempty"`
	TicketInfo       interface{}         `json:"from_ticket_info,omitempty"`
	FaceMonitorToken string              `json:"face_monitor_token,omitempty"`
	SSHCertificate   *SSHCertificateInfo `json:"ssh_certificate,omitempty"`

	Code   string `json:"code"`
	Detail string `json:"detail"`
	Error  string `json:"error"`
}

type SSHCertificateInfo struct {
	SerialNumber         string `json:"serial_number"`
	LeaseDuration        int    `json:"lease_duration"`
	KeyID                string `json:"key_id"`
	Principal            string `json:"principal"`
	PublicKeyFingerprint string `json:"public_key_fingerprint"`
}

func (c *ConnectToken) ClearSSHCertificatePrivateKey() {
	c.Account.ClearSSHCertificatePrivateKey()
}

func (c *ConnectToken) ClearSSHCertificateCredential() {
	c.Account.ClearSSHCertificateCredential()
}

func (c *ConnectToken) CreateSession(addr string,
	loginFrom, SessionType LabelField) Session {
	return Session{
		User:      c.User.String(),
		Asset:     c.Asset.String(),
		Account:   c.Account.String(),
		Protocol:  c.Protocol,
		OrgID:     c.OrgId,
		UserID:    c.User.ID,
		AssetID:   c.Asset.ID,
		AccountID: c.Account.ID,
		DateStart: common.NewNowUTCTime(),

		RemoteAddr: addr,
		LoginFrom:  loginFrom,
		Type:       SessionType,
		ErrReason:  LabelField(SessionReplayErrUnsupported),
		TokenId:    c.Id,
	}
}

type ConnectTokenInfo struct {
	ID                   string          `json:"id"`
	Value                string          `json:"value"`
	ExpireTime           int             `json:"expire_time"`
	Protocol             string          `json:"protocol"`
	Account              string          `json:"account,omitempty"`
	User                 *ObjectId       `json:"user,omitempty"`
	Asset                *ObjectId       `json:"asset,omitempty"`
	InputUsername        string          `json:"input_username,omitempty"`
	InputSecretType      string          `json:"input_secret_type,omitempty"`
	PersonalCredentialID *string         `json:"personal_credential_id,omitempty"`
	ConnectMethod        string          `json:"connect_method,omitempty"`
	ConnectOptions       ConnectOptions  `json:"connect_options,omitempty"`
	Actions              Actions         `json:"actions,omitempty"`
	IsActive             *bool           `json:"is_active,omitempty"`
	IsReusable           *bool           `json:"is_reusable,omitempty"`
	IsExpired            *bool           `json:"is_expired,omitempty"`
	DateExpired          *common.UTCTime `json:"date_expired,omitempty"`
	OrgID                string          `json:"org_id,omitempty"`
	OrgName              string          `json:"org_name,omitempty"`
	RemoteAddr           string          `json:"remote_addr,omitempty"`
	FaceToken            string          `json:"face_token,omitempty"`
	FaceMonitorToken     string          `json:"face_monitor_token,omitempty"`

	Ticket     *ObjectId  `json:"from_ticket,omitempty"`
	TicketInfo TicketInfo `json:"from_ticket_info,omitempty"`

	Code   string `json:"code,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type ClipboardPolicy struct {
	Copy  *ClipboardPolicyItem `json:"copy"`
	Paste *ClipboardPolicyItem `json:"paste"`
}

type ClipboardPolicyItem struct {
	Enabled       bool    `json:"enabled"`
	Action        string  `json:"action"`
	PermAllowed   bool    `json:"perm_allowed"`
	ACLAction     *string `json:"acl_action"`
	TextLimit     int     `json:"text_limit"`
	FileSizeLimit int     `json:"file_size_limit"`
}

const (
	ACLReview = "acl_review"
	ACLReject = "acl_reject"

	ACLFaceVerify             = "acl_face_verify"
	ACLFaceOnline             = "acl_face_online"
	ACLFaceOnlineNotSupported = "acl_face_online_not_supported"
)

type ConnectOptions struct {
	Charset            *string `json:"charset,omitempty"`
	DisableAutoHash    *bool   `json:"disableautohash,omitempty"`
	BackspaceAsCtrlH   *bool   `json:"backspaceAsCtrlH,omitempty"`
	UseSysDBA          bool    `json:"use_sysdba,omitempty"`
	Resolution         string  `json:"resolution"`
	RemoteMicrophone   *bool   `json:"remote_microphone,omitempty"`
	RDPConnectionSpeed string  `json:"rdp_connection_speed,omitempty"`

	FilenameConflictResolution string `json:"file_name_conflict_resolution,omitempty"`
	TerminalThemeName          string `json:"terminal_theme_name,omitempty"`
	Language                   string `json:"lang,omitempty"`
}

type ConnectMethod struct {
	Component string `json:"component"`
	Type      string `json:"type"`
	Label     string `json:"label"`
	Value     string `json:"value"`
}

// token 授权和过期状态

type TokenCheckStatus struct {
	Detail  string `json:"detail"`
	Code    string `json:"code"`
	Expired bool   `json:"expired"`
}

const (
	CodePermOk             = "perm_ok"
	CodePermAccountInvalid = "perm_account_invalid"
	CodePermExpired        = "perm_expired"
)

const ConnectApplet = "applet"
