package main

import (
	"github.com/bmatcuk/go-vagrant"
	"github.com/hashicorp/terraform/helper/schema"

	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func resourceVagrantVM() *schema.Resource {
	return &schema.Resource{
		Create: resourceVagrantVMCreate,
		Read:   resourceVagrantVMRead,
		Update: resourceVagrantVMUpdate,
		Delete: resourceVagrantVMDelete,
		Exists: resourceVagrantVMExists,

		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"vagrantfile_dir": &schema.Schema{
				Description:  "Path to the directory where the Vagrantfile can be found. Defaults to the current directory.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      ".",
				ValidateFunc: resourceVagrantVMPathToVagrantfileValidate,
			},

			"ssh_config": &schema.Schema{
				Description: "SSH connection information.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": &schema.Schema{
							Description: "Connection type. Only valid option is ssh at this time.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"user": &schema.Schema{
							Description: "The user for the connection.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"host": &schema.Schema{
							Description: "The address of the resource to connect to.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"port": &schema.Schema{
							Description: "The port to connect to.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"private_key": &schema.Schema{
							Description: "Private SSH key for the connection.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"agent": &schema.Schema{
							Description: "Whether or not to use the agent to authenticate.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func resourceVagrantVMCreate(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutCreate))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	cmd := client.Up()
	cmd.Context = ctx
	if err := cmd.Run(); err != nil {
		return err
	}

	d.SetId(buildId(cmd.VMInfo))

	return readVagrantInfo(client, ctx, d)
}

func resourceVagrantVMRead(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutRead))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	return readVagrantInfo(client, ctx, d)
}

func resourceVagrantVMUpdate(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutUpdate))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	cmd := client.Reload()
	cmd.Context = ctx
	if err := cmd.Run(); err != nil {
		return nil
	}

	return readVagrantInfo(client, ctx, d)
}

func resourceVagrantVMDelete(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutDelete))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	cmd := client.Destroy()
	cmd.Context = ctx
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func resourceVagrantVMExists(d *schema.ResourceData, m interface{}) (bool, error) {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutRead))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return false, err
	}

	cmd := client.Status()
	cmd.Context = ctx
	if err := cmd.Run(); err != nil {
		return false, err
	}

	// if any machines are not running, then let's say they don't exist
	exists := true
	for _, status := range cmd.Status {
		if status != "running" {
			exists = false
			break
		}
	}

	if !exists {
		// this will force terraform to run create again
		d.SetId("")
	}

	return exists, nil
}

func resourceVagrantVMPathToVagrantfileValidate(val interface{}, key string) ([]string, []error) {
	path := filepath.Join(val.(string), "Vagrantfile")
	if _, err := os.Stat(path); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func getContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	if int64(timeout) > 0 {
		return context.WithTimeout(ctx, timeout)
	}
	return ctx, func() {}
}

func buildId(info map[string]*vagrant.VMInfo) string {
	var keys sort.StringSlice = make([]string, len(info)+1)
	keys[0] = "vagrant"
	i := 1
	for key, _ := range info {
		keys[i] = key
		i++
	}
	keys[1:].Sort()
	return strings.Join(keys, ":")
}

func readVagrantInfo(client *vagrant.VagrantClient, ctx context.Context, d *schema.ResourceData) error {
	cmd := client.SSHConfig()
	cmd.Context = ctx
	if err := cmd.Run(); err != nil {
		return err
	}

	sshConfigs := make([]map[string]string, len(cmd.Configs))
	i := 0
	for _, config := range cmd.Configs {
		sshConfig := make(map[string]string, 6)
		sshConfig["type"] = "ssh"
		sshConfig["user"] = config.User
		sshConfig["host"] = config.HostName
		sshConfig["port"] = strconv.Itoa(config.Port)
		if privateKey, err := ioutil.ReadFile(config.IdentityFile); err == nil {
			sshConfig["private_key"] = string(privateKey)
		}
		sshConfig["agent"] = "false"
		sshConfigs[i] = sshConfig
		i++
	}
	d.Set("ssh_config", sshConfigs)

	if len(sshConfigs) == 1 {
		d.SetConnInfo(sshConfigs[0])
	}

	return nil
}
