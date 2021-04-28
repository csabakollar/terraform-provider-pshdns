package pshdns

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/crypto/ssh"
)

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func waitForLock(client *clientInfo) bool {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	time.Sleep(time.Duration(r.Intn(100)) * time.Millisecond)

	locked := fileExists(client.lockfile)

	for locked == true {
		time.Sleep(100 * time.Millisecond)
		locked = fileExists(client.lockfile)
	}

	time.Sleep(1000 * time.Millisecond)
	return true
}

func runRemoteCommand(c *clientInfo, cmd string) ([]byte, error) {
	conn, err := ssh.Dial("tcp", c.sshServer+":"+c.sshPort, c.sshConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return []byte{}, err
	}

	output, err := session.Output(cmd)
	if err != nil {
		return output, fmt.Errorf("failed to execute command '%s' on server: %v", cmd, err)
	}

	return output, err
}

func resourcePshDNS() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourcePshDNSRecordCreate,
		ReadContext:   resourcePshDNSRecordRead,
		DeleteContext: resourcePshDNSRecordDelete,
		Description:   "Resource representing a single DNS record entry",

		Schema: map[string]*schema.Schema{
			"zone_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the zone where the DNS record will be created/managed.",
			},
			"record_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Value for the record, which will be created.",
			},
			"record_type": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Type of the record, currently only supporting A, PTR, CNAME types.",
			},
			"ipv4_address": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "If record type is set to A, this is a mandatory field. Specify the IP address you wish the record to point to.",
			},
			"hostname_alias": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "If record type is set to CNAME, this is a mandatory field. Specify the hostname you wish the record to point to.",
			},
			"ptr_domainname": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "If record type is set to PTR, this is a mandatory field. Specify the hostname you wish the record to point to.",
			},
		},
	}
}

func resourcePshDNSRecordCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	//convert the interface so we can use the variables like username, etc
	client := m.(*clientInfo)

	zoneName := d.Get("zone_name").(string)
	recordType := d.Get("record_type").(string)
	recordName := d.Get("record_name").(string)
	ipv4address := d.Get("ipv4_address").(string)
	hostnamealias := d.Get("hostname_alias").(string)
	ptrdomainname := d.Get("ptr_domainname").(string)

	var id string = zoneName + "_" + recordName + "_" + recordType

	var psCommand string

	waitForLock(client)

	file, err := os.Create(client.lockfile)
	if err != nil {
		return diag.FromErr(err)
	}

	switch recordType {
	case "A":
		if ipv4address == "" {
			return diag.FromErr(errors.New("Must provide ipv4address if record_type is 'A'"))
		}
		psCommand = "Add-DNSServerResourceRecordA -ZoneName " + zoneName + " -Name " + recordName + " -IPv4Address " + ipv4address
	case "CNAME":
		if hostnamealias == "" {
			return diag.FromErr(errors.New("Must provide hostnamealias if record_type is 'CNAME'"))
		}
		psCommand = "Add-DNSServerResourceRecord -ZoneName " + zoneName + " -" + recordType + " -Name " + recordName + " -HostNameAlias " + hostnamealias
	case "PTR":
		if ptrdomainname == "" {
			return diag.FromErr(errors.New("Must provide ptrdomainname if record_type is 'PTR'"))
		}
		psCommand = "Add-DNSServerResourceRecord -ZoneName " + zoneName + " -" + recordType + " -Name " + recordName + " -PtrDomainName " + ptrdomainname
	default:
		return diag.FromErr(errors.New("Unknown record type. This provider currently only supports 'A', 'CNAME', and 'PTR' records"))
	}

	if client.dnsServer != "" {
		psCommand = psCommand + " -ComputerName " + client.dnsServer
	}

	_, err = runRemoteCommand(client, psCommand)
	if err != nil {
		return diag.FromErr(err)
	}
	// log.Println(out)
	d.SetId(id)

	file.Close()
	os.Remove(client.lockfile)

	return diags
}

func resourcePshDNSRecordRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	//convert the interface so we can use the variables like username, etc
	client := m.(*clientInfo)

	var diags diag.Diagnostics

	zoneName := d.Get("zone_name").(string)
	recordType := d.Get("record_type").(string)
	recordName := d.Get("record_name").(string)

	psCommand := "try { $record = Get-DnsServerResourceRecord -ZoneName " + zoneName + " -RRType " + recordType + " -Name " + recordName + " -ErrorAction Stop } catch { $record = '''' }; if ($record) { write-host 'RECORD_FOUND' }"

	if client.dnsServer != "" {
		psCommand = psCommand + " -ComputerName " + client.dnsServer
	}

	_, err := runRemoteCommand(client, psCommand)
	if err != nil {
		if !strings.Contains(err.Error(), "ObjectNotFound") {
			//something bad happened
			return diag.FromErr(err)
		}
		//not able to find the record - this is an error but ok
		d.SetId("")
		return nil
	}

	var id string = zoneName + "_" + recordName + "_" + recordType
	d.SetId(id)

	return diags
}

func resourcePshDNSRecordDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	//convert the interface so we can use the variables like username, etc
	client := m.(*clientInfo)

	var diags diag.Diagnostics

	waitForLock(client)

	file, err := os.Create(client.lockfile)
	if err != nil {
		return diag.FromErr(err)
	}

	zoneName := d.Get("zone_name").(string)
	recordType := d.Get("record_type").(string)
	recordName := d.Get("record_name").(string)

	psCommand := "Remove-DNSServerResourceRecord -ZoneName " + zoneName + " -RRType " + recordType + " -Name " + recordName + " -Confirm:$false -Force"

	if client.dnsServer != "" {
		psCommand = psCommand + " -ComputerName " + client.dnsServer
	}

	out, err := runRemoteCommand(client, psCommand)
	if err != nil {
		return diag.FromErr(err)
	}
	log.Println(out)

	// d.SetId("") is automatically called assuming delete returns no errors, but it is added here for explicitness.
	d.SetId("")

	file.Close()
	os.Remove(client.lockfile)

	return diags
}
