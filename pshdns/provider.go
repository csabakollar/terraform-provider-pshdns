package pshdns

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/crypto/ssh"
)

func init() {
	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	// schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
	// 	desc := s.Description
	// 	if s.Default != nil {
	// 		desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
	// 	}
	// 	return strings.TrimSpace(desc)
	// }
}

// Provider allows making changes to Windows DNS server
// Utilises Powershell to connect to domain controller
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PSH_USERNAME", nil),
				Description: "Username to connect to AD.",
			},
			"password": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PSH_PASSWORD", nil),
				Description: "The password to connect to AD.",
			},
			"ssh_server": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("PSH_SSH_SERVER", nil),
				Description: "The SSH server to connect to.",
			},
			"ssh_port": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PSH_SSH_SERVER_PORT", "22"),
				Description: "The SSH server port to connect to.",
			},
			"dns_server": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("PSH_DNS_SERVER", ""),
				Description: "The DNS server where the zone is hosted.",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"pshdns": resourcePshDNS(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

type clientInfo struct {
	sshConfig *ssh.ClientConfig
	sshServer string
	sshPort   string
	dnsServer string
	lockfile  string
}

func providerConfigure(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	username := d.Get("username").(string)
	password := d.Get("password").(string)
	sshServer := d.Get("ssh_server").(string)
	sshPort := d.Get("ssh_port").(string)
	dnsServer := d.Get("dns_server").(string)

	if username == "" || password == "" || sshServer == "" {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Missing mandatory parameter",
			Detail:   "You must specify the username, password and ssh_server parameters to be able to connect to the ssh server",
		})
		return nil, diags
	}

	f, _ := ioutil.TempFile("", "terraform-pshdns")
	lockFile := f.Name()
	os.Remove(f.Name())

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client := clientInfo{
		sshConfig: sshConfig,
		sshServer: sshServer,
		sshPort:   sshPort,
		dnsServer: dnsServer,
		lockfile:  lockFile,
	}

	return &client, diags
}
