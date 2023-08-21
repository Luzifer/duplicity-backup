package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Luzifer/go_helpers/str"
	"github.com/pkg/errors"
)

const notifyRequestTimeout = 2 * time.Second

func (c *configFile) Notify(command string, success bool, err error) error {
	if !str.StringInSlice(command, notifyCommands) {
		return nil
	}

	errs := []error{}

	for _, n := range []func(bool, error) error{
		c.notifyMonDash,
		c.notifySlack,
	} {
		if e := n(success, err); e != nil {
			errs = append(errs, e)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	estr := ""
	for _, e := range errs {
		if e == nil {
			continue
		}

		estr = fmt.Sprintf("%s\n- %s", estr, e)
	}
	return errors.Errorf("%d notifiers failed:%s", len(errs), estr)
}

type mondashResult struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Freshness   int64  `json:"freshness"`
	IgnoreMAD   bool   `json:"ignore_mad"`
	HideMAD     bool   `json:"hide_mad"`
	HideValue   bool   `json:"hide_value"`
}

//revive:disable-next-line:flag-parameter // not a flag parameter
func (c *configFile) notifyMonDash(success bool, err error) error {
	if c.Notifications.MonDash.BoardURL == "" {
		return nil
	}

	monitoringResult := mondashResult{
		Title:     fmt.Sprintf("duplicity-backup on %s", c.Hostname),
		Freshness: c.Notifications.MonDash.Freshness,
		IgnoreMAD: true,
		HideMAD:   true,
		HideValue: true,
	}

	if success {
		monitoringResult.Status = "OK"
		monitoringResult.Description = "Backup succeeded"
	} else {
		monitoringResult.Status = "Critical"
		monitoringResult.Description = fmt.Sprintf("Backup failed: %s", err)
	}

	buf := bytes.NewBuffer([]byte{})
	if err = json.NewEncoder(buf).Encode(monitoringResult); err != nil {
		return errors.Wrap(err, "encoding request payload")
	}

	url := fmt.Sprintf("%s/duplicity-%s",
		c.Notifications.MonDash.BoardURL,
		c.Hostname,
	)

	ctx, cancel := context.WithTimeout(context.Background(), notifyRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, buf)
	if err != nil {
		return errors.Wrap(err, "creating request")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Notifications.MonDash.Token)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing request")
	}
	defer res.Body.Close() //nolint:errcheck // Will be cleaned by process exit shortly after

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

type slackResult struct {
	Username string `json:"username,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Icon     string `json:"icon_emoji,omitempty"`
	Text     string `json:"text"`
}

//revive:disable-next-line:flag-parameter // not a flag parameter
func (c *configFile) notifySlack(success bool, err error) error {
	if c.Notifications.Slack.HookURL == "" {
		return nil
	}

	text := "Backup succeeded"
	if !success {
		text = fmt.Sprintf("Backup failed: %s", err)
	}

	sr := slackResult{
		Username: c.Notifications.Slack.Username,
		Channel:  c.Notifications.Slack.Channel,
		Icon:     c.Notifications.Slack.Emoji,
		Text:     text,
	}

	buf := bytes.NewBuffer([]byte{})
	if err = json.NewEncoder(buf).Encode(sr); err != nil {
		return errors.Wrap(err, "encoding payload")
	}

	ctx, cancel := context.WithTimeout(context.Background(), notifyRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Notifications.Slack.HookURL, buf)
	if err != nil {
		return errors.Wrap(err, "creating request")
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing request")
	}
	defer res.Body.Close() //nolint:errcheck // Will be cleaned by process exit shortly after

	if res.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}
