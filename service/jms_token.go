package service

import (
	"fmt"
	"strings"

	"github.com/jumpserver-dev/sdk-go/model"
)

func (s *JMService) GetConnectTokenInfo(tokenId string, expireNow bool) (resp model.ConnectToken, err error) {
	return s.GetConnectTokenInfoWithPublicKey(tokenId, expireNow, "")
}

// GetConnectTokenInfoWithPublicKey retrieves connection credentials and sends
// an ephemeral SSH public key when the caller can use SSH certificates.
func (s *JMService) GetConnectTokenInfoWithPublicKey(tokenId string, expireNow bool,
	publicKey string) (resp model.ConnectToken, err error) {
	return s.GetConnectTokenInfoWithOptions(tokenId, ConnectTokenSecretOptions{
		ExpireNow: &expireNow, PublicKey: publicKey,
	})
}

type ConnectTokenSecretOptions struct {
	ExpireNow *bool  `json:"expire_now,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
	// ExtraFields supplies future JSON fields; explicitly set typed fields win.
	ExtraFields map[string]any `json:"-"`
}

func (s *JMService) GetConnectTokenInfoWithOptions(tokenId string,
	options ConnectTokenSecretOptions) (resp model.ConnectToken, err error) {
	request := struct {
		ID string `json:"id"`
		ConnectTokenSecretOptions
	}{ID: tokenId, ConnectTokenSecretOptions: options}
	data, err := requestWithExtra(request, options.ExtraFields)
	if err != nil {
		return resp, err
	}
	_, err = s.authClient.Post(SuperConnectTokenSecretURL, data, &resp)
	return
}

func (s *JMService) CreateSuperConnectToken(data *SuperConnectTokenReq) (resp model.ConnectTokenInfo, err error) {
	ak := s.opt.accessKey
	apiClient := s.authClient.Clone()
	if s.opt.sign != nil {
		apiClient.SetAuthSign(s.opt.sign)
	}
	apiClient.SetHeader(orgHeaderKey, orgHeaderValue)
	// 移除 Secret 中的 "-", 保证长度为 32
	secretKey := strings.ReplaceAll(ak.Secret, "-", "")
	encryptKey, err1 := GenerateEncryptKey(secretKey)
	if err1 != nil {
		return resp, err1
	}
	signKey := fmt.Sprintf("%s:%s", ak.ID, encryptKey)
	apiClient.SetHeader(svcHeader, fmt.Sprintf("Sign %s", signKey))
	body, err := requestWithExtra(data, data.ExtraFields)
	if err != nil {
		return resp, err
	}
	_, err = apiClient.Post(SuperConnectTokenInfoURL, body, &resp, data.Params)
	return
}

type SuperConnectTokenReq struct {
	UserId                    string         `json:"user"`
	AssetId                   string         `json:"asset"`
	Account                   string         `json:"account"`
	Protocol                  string         `json:"protocol"`
	ConnectMethod             string         `json:"connect_method"`
	InputUsername             string         `json:"input_username"`
	InputSecret               string         `json:"input_secret"`
	RemoteAddr                string         `json:"remote_addr"`
	InputSecretType           string         `json:"input_secret_type,omitempty"`
	ConnectOptions            map[string]any `json:"connect_options,omitempty"`
	PersonalCredentialID      string         `json:"personal_credential_id,omitempty"`
	PersonalCredentialVersion *int           `json:"personal_credential_version,omitempty"`
	SavePersonalCredential    *bool          `json:"save_personal_credential,omitempty"`
	IsActive                  *bool          `json:"is_active,omitempty"`
	IsReusable                *bool          `json:"is_reusable,omitempty"`

	// ExtraFields supplies future JSON fields; explicitly set typed fields win.
	// Personal credentials still require Core's owner authentication checks.
	ExtraFields map[string]any `json:"-"`

	Params map[string]string `json:"-"`
}

func (s *JMService) GetConnectTokenAppletOption(tokenId string) (resp model.AppletOption, err error) {
	data := map[string]string{
		"id": tokenId,
	}
	_, err = s.authClient.Post(SuperConnectTokenAppletOptionURL, data, &resp)
	return
}

func (s *JMService) ReleaseAppletAccount(accountId string) (err error) {
	data := map[string]string{
		"id": accountId,
	}
	_, err = s.authClient.Post(SuperConnectAppletHostAccountReleaseURL, data, nil)
	return
}

func (s *JMService) GetConnectTokenVirtualAppOption(tokenId string) (resp model.VirtualApp, err error) {
	data := map[string]string{
		"id": tokenId,
	}
	_, err = s.authClient.Post(SuperConnectTokenVirtualAppOptionURL, data, &resp)
	return
}

func (s *JMService) CheckTokenStatus(tokenId string) (res model.TokenCheckStatus, err error) {
	reqURL := fmt.Sprintf(SuperConnectTokenCheckURL, tokenId)
	_, err = s.authClient.Get(reqURL, &res)
	return
}

func (s *JMService) RenewalToken(token string) (resp TokenRenewalResponse, err error) {
	data := map[string]string{
		"id": token,
	}
	_, err = s.authClient.Patch(SuperTokenRenewalURL, data, &resp)
	return
}

type TokenRenewalResponse struct {
	Ok  bool   `json:"ok"`
	Msg string `json:"msg"`
}

func (s *JMService) GetAccountsChat() (resp model.AccountChat, err error) {
	/*
		for AI Chat Account
	*/
	_, err = s.authClient.Get(AccountsChatURL, &resp)
	return
}
