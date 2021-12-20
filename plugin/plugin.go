// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package plugin

import (
	"context"
	"errors"
	"regexp"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone-go/plugin/secret"
	"github.com/sirupsen/logrus"

	"github.com/hashicorp/vault/api"
)

// New returns a new secret plugin that sources secrets
// from the AWS secrets manager.
func New(client *api.Client) secret.Plugin {
	return &plugin{
		client: client,
	}
}

type plugin struct {
	client *api.Client
}

func (p *plugin) Find(ctx context.Context, req *secret.Request) (*drone.Secret, error) {
	path := req.Path
	name := req.Name
	reg, _ := regexp.Compile(`^v(\d+):`)
	version := reg.FindStringSubmatch(path)
	path = reg.ReplaceAllString(path, "")
	if name == "" {
		name = "value"
	}
	_version := "1"
	if len(version) == 2 {
		_version = version[1]
	}
	logrus.WithFields(logrus.Fields{
		"version": _version,
		"path":    path,
		"name":    name,
	}).Debugln("find secret")
	// makes an api call to the aws secrets manager and attempts
	// to retrieve the secret at the requested path.
	params, err := p.find(path, version)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"version": _version,
			"path":    path,
			"name":    name,
		}).Errorln(err)
		return nil, err
	}
	value, ok := params[name]
	if !ok {
		err := errors.New("secret key not found")
		logrus.WithFields(logrus.Fields{
			"version": _version,
			"path: ":  path,
			"name":    name,
			"params":  params,
		}).Errorln(err)
		return nil, err
	}

	// the user can filter out requets based on event type
	// using the X-Drone-Events secret key. Check for this
	// user-defined filter logic.
	events := extractEvents(params)
	if !match(req.Build.Event, events) {
		err := errors.New("access denied: event does not match")
		logrus.WithField("events: ", events).Errorln(err)
		return nil, err
	}

	// the user can filter out requets based on repository
	// using the X-Drone-Repos secret key. Check for this
	// user-defined filter logic.
	repos := extractRepos(params)
	if !match(req.Repo.Slug, repos) {
		err := errors.New("access denied: repository does not match")
		logrus.WithField("repos: ", repos).Errorln(err)
		return nil, err
	}

	// the user can filter out requets based on repository
	// branch using the X-Drone-Branches secret key. Check
	// for this user-defined filter logic.
	branches := extractBranches(params)
	if !match(req.Build.Target, branches) {
		err := errors.New("access denied: branch does not match")
		logrus.WithField("branches: ", branches).Errorln(err)
		return nil, err
	}

	return &drone.Secret{
		Name: name,
		Data: value,
		Pull: true, // always true. use X-Drone-Events to prevent pull requests.
		Fork: true, // always true. use X-Drone-Events to prevent pull requests.
	}, nil
}

// helper function returns the secret from vault.
func (p *plugin) find(path string, version []string) (map[string]string, error) {
	values := map[string][]string{}
	if len(version) == 2 {
		values = map[string][]string{
			"version": {version[1]},
		}
	}

	secret, err := p.client.Logical().ReadWithData(path, values)
	if err != nil {
		return nil, err
	}
	if secret == nil || secret.Data == nil {
		return nil, errors.New("secret not found")
	}

	// HACK: the vault v2 key value store is confusing
	// and I could not quite figure out how to work with
	// the api. This is the workaround I came up with.
	v := secret.Data["data"]
	if data, ok := v.(map[string]interface{}); ok {
		secret.Data = data
	}

	params := map[string]string{}
	for k, v := range secret.Data {
		s, ok := v.(string)
		if !ok {
			continue
		}
		params[k] = s
	}
	return params, err
}
