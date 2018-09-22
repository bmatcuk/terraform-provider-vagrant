package main

import (
	"github.com/bmatcuk/go-vagrant"
	"github.com/hashicorp/terraform/helper/schema"

	"context"
	"fmt"
	"io/ioutil"
	"log"
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
			"vagrantfile_dir": {
				Description:  "Path to the directory where the Vagrantfile can be found. Defaults to the current directory.",
				Type:         schema.TypeString,
				Optional:     true,
				Default:      ".",
				ValidateFunc: resourceVagrantVMPathToVagrantfileValidate,
			},

			"env": {
				Description: "Environment variables to pass to the Vagrantfile.",
				Type:        schema.TypeMap,
				Optional:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"machine_names": {
				Description: "Names of the vagrant machines from the Vagrantfile. Names are in the same order as ssh_config.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"ssh_config": {
				Description: "SSH connection information.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Description: "Connection type. Only valid option is ssh at this time.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"user": {
							Description: "The user for the connection.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"host": {
							Description: "The address of the resource to connect to.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"port": {
							Description: "The port to connect to.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"private_key": {
							Description: "Private SSH key for the connection.",
							Type:        schema.TypeString,
							Computed:    true,
						},

						"agent": {
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

	log.Println("Bringing up vagrant...")
	cmd := client.Up()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Parallel = true
	cmd.Verbose = true
	if err := cmd.Run(); err != nil {
		return err
	}

	d.SetId(buildId(cmd.VMInfo))

	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMRead(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutRead))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMUpdate(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutUpdate))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	env := buildEnvironment(d.Get("env").(map[string]interface{}))

	// reload will halt any running machines, then destroy any halted or
	// suspended machines, and bring them back up
	log.Println("Reloading vagrant...")
	reload := client.Reload()
	reload.Context = ctx
	reload.Env = env
	reload.Verbose = true
	if err := reload.Run(); err != nil {
		return err
	}

	// reload will not bring up new machines, so bring them up here...
	log.Println("Checking machine states...")
	status := client.Status()
	status.Context = ctx
	status.Env = env
	status.Verbose = true
	if err := status.Run(); err != nil {
		return err
	}

	allExist := true
	for _, state := range status.Status {
		if state == "not_created" {
			allExist = false
			break
		}
	}

	if !allExist {
		log.Println("Bringing up new machines...")
		up := client.Up()
		up.Context = ctx
		up.Env = env
		up.Parallel = true
		up.Verbose = true
		if err := up.Run(); err != nil {
			return err
		}
	}

	// we're done!
	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMDelete(d *schema.ResourceData, m interface{}) error {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutDelete))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return err
	}

	log.Println("Destroying vagrant...")
	cmd := client.Destroy()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Verbose = true
	return cmd.Run()
}

func resourceVagrantVMExists(d *schema.ResourceData, m interface{}) (bool, error) {
	ctx, cancelFunc := getContext(d.Timeout(schema.TimeoutRead))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return false, err
	}

	log.Println("Getting vagrant status...")
	cmd := client.Status()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Verbose = true
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
	for key := range info {
		keys[i] = key
		i++
	}
	keys[1:].Sort()
	return strings.Join(keys, ":")
}

func readVagrantInfo(ctx context.Context, client *vagrant.VagrantClient, d *schema.ResourceData) error {
	log.Println("Getting vagrant ssh-config...")
	cmd := client.SSHConfig()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Verbose = true
	if err := cmd.Run(); err != nil {
		return err
	}

	sshConfigs := make([]map[string]string, len(cmd.Configs))
	keys := make([]string, len(cmd.Configs))
	i := 0
	for key, config := range cmd.Configs {
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
		keys[i] = key
		i++
	}

	d.Set("ssh_config", sshConfigs)
	d.Set("machine_names", keys)

	if len(sshConfigs) == 1 {
		d.SetConnInfo(sshConfigs[0])
	}

	return nil
}

func buildEnvironment(env map[string]interface{}) []string {
	if len(env) == 0 {
		return nil
	}

	envArray := make([]string, len(env))
	i := 0
	for key, value := range env {
		envArray[i] = fmt.Sprintf("%v=%v", key, value)
		i++
	}

	log.Println("Environment:", envArray)
	return envArray
}
