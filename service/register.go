package service

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jumpserver-dev/sdk-go/httplib"
	"github.com/jumpserver-dev/sdk-go/model"
)

func RegisterTerminalAccount(coreHost, componentName, name, token string) (res model.Terminal, err error) {
	return RegisterTerminalAccountWithOptions(coreHost, componentName, name, token, TerminalRegistrationOptions{})
}

type TerminalRegistrationOptions struct {
	ProviderID string `json:"provider_id,omitempty"`
	// ExtraFields supplies future JSON fields; explicitly set typed fields win.
	ExtraFields map[string]any `json:"-"`
}

func terminalRegistrationRequest(name, componentName string, options TerminalRegistrationOptions) (any, error) {
	request := struct {
		Name    string `json:"name"`
		Comment string `json:"comment"`
		Type    string `json:"type"`
		TerminalRegistrationOptions
	}{Name: name, Comment: componentName, Type: componentName, TerminalRegistrationOptions: options}
	return requestWithExtra(request, options.ExtraFields)
}

func RegisterTerminalAccountWithOptions(coreHost, componentName, name, token string,
	options TerminalRegistrationOptions) (res model.Terminal, err error) {
	client, err := httplib.NewClient(coreHost, time.Second*30)
	if err != nil {
		return model.Terminal{}, err
	}
	client.SetHeader("Authorization", fmt.Sprintf("BootstrapToken %s", token))
	data, err := terminalRegistrationRequest(name, componentName, options)
	if err != nil {
		return res, err
	}
	_, err = client.Post(TerminalRegisterURL, data, &res)
	return
}

func ValidAccessKey(coreHost string, key model.AccessKey) error {
	client, err := httplib.NewClient(coreHost, time.Second*30)
	if err != nil {
		return err
	}
	sign := httplib.SigAuth{
		KeyID:    key.ID,
		SecretID: key.Secret,
	}
	client.SetAuthSign(&sign)
	var (
		user model.User
		res  *http.Response
	)

	res, err = client.Get(UserProfileURL, &user)
	if err != nil {
		if res == nil {
			return fmt.Errorf("%w:%s", ErrConnect, err.Error())
		}
		if res.StatusCode == http.StatusUnauthorized {
			return ErrUnauthorized
		}
		return fmt.Errorf("%w: %s", ErrInvalid, err.Error())
	}
	if user.ID == "" {
		return ErrInvalid
	}
	return nil
}

var (
	ErrConnect      = errors.New("connect failed")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalid      = errors.New("invalid user")
)
