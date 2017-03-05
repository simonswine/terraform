package git

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"

	"golang.org/x/crypto/ssh"
	"srcd.works/go-git.v4"
	gitplumbing "srcd.works/go-git.v4/plumbing"
	gitssh "srcd.works/go-git.v4/plumbing/transport/ssh"
)

const DefaultRemoteName = "origin"

func backendSchema() *schema.Backend {
	return &schema.Backend{
		Schema: map[string]*schema.Schema{
			"path": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to store state in Git",
			},

			"url": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Git repo URL",
			},

			"branch": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Git branch",
				Default:     "master",
			},

			"ssh_key_path": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to a SSH Private Key",
				Default:     "", // To prevent input
			},

			"ssh_user": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "SSH user",
				Default:     "git",
			},
		},
	}
}

// New creates a new backend for Git remote state.
func New() backend.Backend {
	result := &Backend{Backend: backendSchema()}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	configData *schema.ResourceData
}

func (b *Backend) configure(ctx context.Context) error {
	// Grab the resource data
	b.configData = schema.FromContextBackendConfig(ctx)

	// Initialize a client to test config
	_, err := b.clientRaw()
	return err
}

func (b *Backend) clientRaw() (*git.CloneOptions, error) {
	data := b.configData

	options := &git.CloneOptions{
		SingleBranch: true,
		RemoteName:   DefaultRemoteName,
	}

	// Read URL
	if v, ok := data.GetOk("url"); ok && v.(string) != "" {
		options.URL = v.(string)
	}

	// Read branch
	if v, ok := data.GetOk("branch"); ok && v.(string) != "" {
		options.ReferenceName = gitplumbing.ReferenceName(
			fmt.Sprintf("refs/heads/%s", v.(string)),
		)
	}

	// Load SSH private key if needed
	if v, ok := data.GetOk("ssh_key_path"); ok && v.(string) != "" {
		sshKeyFd, err := os.Open(v.(string))
		if err != nil {
			return nil, fmt.Errorf("error opening ssh key: ", err)
		}
		defer sshKeyFd.Close()

		sshKeyBuffer, err := ioutil.ReadAll(sshKeyFd)
		if err != nil {
			return nil, fmt.Errorf("error reading ssh key: ", err)
		}

		signer, err := ssh.ParsePrivateKey(sshKeyBuffer)
		if err != nil {
			return nil, fmt.Errorf("error parsing ssh key: ", err)
		}

		sshAuth := &gitssh.PublicKeys{
			Signer: signer,
		}

		if v, ok := data.GetOk("ssh_user"); ok && v.(string) != "" {
			sshAuth.User = v.(string)
		}

		options.Auth = sshAuth
	}

	return options, nil
}
