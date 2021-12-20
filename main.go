package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/drone/drone-go/plugin/secret"
	"github.com/drone/drone-vault/plugin/token"
	"github.com/drone/drone-vault/plugin/token/approle"
	"github.com/drone/drone-vault/plugin/token/kubernetes"
	"github.com/hashicorp/vault/api"
	"github.com/kelseyhightower/envconfig"
	"github.com/mine-golang/drone-docker-vault/plugin"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// additional vault environment variables that are
// used by the vault client.
var envs = []string{
	"VAULT_ADDR",
	"VAULT_CACERT",
	"VAULT_CAPATH",
	"VAULT_CLIENT_CERT",
	"VAULT_SKIP_VERIFY",
	"VAULT_MAX_RETRIES",
	"VAULT_TOKEN",
	"VAULT_TLS_SERVER_NAME",
}

type config struct {
	Level              string        `envconfig:"DEBUG_LEVEL" default:"info"`
	Address            string        `envconfig:"DRONE_BIND"`
	Secret             string        `envconfig:"DRONE_SECRET"`
	VaultAddr          string        `envconfig:"VAULT_ADDR"`
	VaultToken         string        `envconfig:"VAULT_TOKEN"`
	VaultRenew         time.Duration `envconfig:"VAULT_TOKEN_RENEWAL" default:"84h"`
	VaultTTL           time.Duration `envconfig:"VAULT_TOKEN_TTL" default:"168h"`
	VaultAuthType      string        `envconfig:"VAULT_AUTH_TYPE"`
	VaultAuthMount     string        `envconfig:"VAULT_AUTH_MOUNT_POINT"`
	VaultApproleID     string        `envconfig:"VAULT_APPROLE_ID"`
	VaultApproleSecret string        `envconfig:"VAULT_APPROLE_SECRET"`
	VaultKubeRole      string        `envconfig:"VAULT_KUBERNETES_ROLE"`
}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)
}

func initLog(DebugLevel string) {
	switch DebugLevel {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warning":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		logrus.SetLevel(logrus.PanicLevel)
	}
}

func main() {
	spec := new(config)
	err := envconfig.Process("", spec)
	if err != nil {
		logrus.Fatal(err)
	}
	initLog(spec.Level)

	if err != nil {
		logrus.Fatal(err)
	}
	if spec.Secret == "" {
		logrus.Fatalln("missing secret key")
	}
	if spec.VaultAddr == "" {
		logrus.Warnln("missing vault address")
	}
	if spec.VaultToken == "" {
		logrus.Warnln("missing vault token")
	}
	if spec.Address == "" {
		spec.Address = ":3000"
	}

	// creates the vault client from the VAULT_*
	// environment variables.
	client, err := api.NewClient(nil)
	if err != nil {
		logrus.Fatalln(err)
	}

	// global context
	ctx := context.Background()

	http.Handle("/", secret.Handler(
		spec.Secret,
		plugin.New(client),
		logrus.StandardLogger(),
	))

	var g errgroup.Group

	// the token can be fetched at runtime if an auth
	// provider is configured. otherwise, the user must
	// specify a VAULT_TOKEN.
	if spec.VaultAuthType == kubernetes.Name {
		renewer := kubernetes.NewRenewer(
			client,
			spec.VaultAddr,
			spec.VaultKubeRole,
			spec.VaultAuthMount,
		)
		err := renewer.Renew(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}

		// the vault token needs to be periodically
		// refreshed and the kubernetes token has a
		// max age of 32 days.
		g.Go(func() error {
			return renewer.Run(ctx, spec.VaultRenew)
		})
	} else if spec.VaultAuthType == approle.Name {
		renewer := approle.NewRenewer(
			client,
			spec.VaultApproleID,
			spec.VaultApproleSecret,
			spec.VaultTTL,
		)
		err := renewer.Renew(ctx)
		if err != nil {
			logrus.Fatalln(err)
		}

		// the vault token needs to be periodically refreshed
		g.Go(func() error {
			return renewer.Run(ctx, spec.VaultRenew)
		})
	} else {
		g.Go(func() error {
			return token.NewRenewer(
				client, spec.VaultTTL, spec.VaultRenew).Run(ctx)
		})
	}

	g.Go(func() error {
		logrus.Infof("server listening on address %s", spec.Address)
		return http.ListenAndServe(spec.Address, nil)
	})

	if err := g.Wait(); err != nil {
		logrus.Fatal(err)
	}
}
