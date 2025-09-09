package service

import (
	"fmt"

	"github.com/jumpserver-dev/sdk-go/model"
)

func (s *JMService) CheckUserCookie(cookies map[string]string) (user *model.User, err error) {
	client := s.authClient.Clone()
	for k, v := range cookies {
		client.SetCookie(k, v)
	}
	_, err = client.Get(UserProfileURL, &user)
	return
}

func (s *JMService) GetUserByUsername(username string) (user *model.User, err error) {
	reqURL := fmt.Sprintf(UserDetailURL, username)
	_, err = s.authClient.Get(reqURL, &user)
	return
}
