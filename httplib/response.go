package httplib

import (
	"encoding/json"
	"errors"
	"fmt"
)

func (c *Client) responseDestination(response any) any {
	if c.extraResponse == nil {
		return response
	}
	return &responseTargets{response: response, extra: c.extraResponse}
}

type responseTargets struct {
	response any
	extra    any
}

func (r *responseTargets) UnmarshalJSON(data []byte) (err error) {
	if r.response != nil {
		err = json.Unmarshal(data, r.response)
	}
	// Attempt both destinations so a type error does not discard other fields.
	if extraErr := json.Unmarshal(data, r.extra); extraErr != nil {
		return errors.Join(err, fmt.Errorf("decode extra response: %w", extraErr))
	}
	return err
}
