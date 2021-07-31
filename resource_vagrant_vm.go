package main

import (
	"fmt"
	"os"

	"github.com/bmatcuk/go-vagrant"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"context"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func resourceVagrantVM() *schema.Resource {
	return &schema.Resource{
		Description: "Integrate vagrant into terraform.",

		CreateContext: resourceVagrantVMCreate,
		ReadContext:   resourceVagrantVMRead,
		UpdateContext: resourceVagrantVMUpdate,
		DeleteContext: resourceVagrantVMDelete,

		CustomizeDiff: customdiff.All(
			customdiff.ForceNewIfChange("name", func(ctx context.Context, old, new, meta interface{}) bool {
				return old != new
			}),
		),

		SchemaVersion: 1,
		Schema: map[string]*schema.Schema{
			"name": {
				Description: "Name of the Vagrant resource. Forces resource to destroy/recreate if changed.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "vagrantbox",
			},

			"vagrantfile_dir": {
				Description:  "Path to the directory where the Vagrantfile can be found.",
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

			"get_ports": {
				Description: "Whether or not to retrieve forwarded port information. See `ports`.",
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
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

			"ports": {
				Description: "Forwarded ports per machine. Only set if `get_ports` is true.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Schema{
					Type: schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"guest": {
								Description: "The port on the guest machine.",
								Type:        schema.TypeInt,
								Computed:    true,
							},

							"host": {
								Description: "The port on the host machine which maps to the guest port.",
								Type:        schema.TypeInt,
								Computed:    true,
							},
						},
					},
				},
			},
		},
	}
}

func resourceVagrantVMCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ctx, cancelFunc := contextWithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("Bringing up vagrant...")
	cmd := client.Up()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Parallel = true
	cmd.Verbose = true
	if err := cmd.Run(); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(buildId(cmd.VMInfo))

	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ctx, cancelFunc := contextWithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	exists, err := checkIfVMExists(ctx, client, d)
	if err != nil {
		return diag.FromErr(err)
	} else if !exists {
		// this will force terraform to run create again
		d.SetId("")
		return nil
	}

	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ctx, cancelFunc := contextWithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return diag.FromErr(err)
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
		return diag.FromErr(err)
	}

	// reload will not bring up new machines, so bring them up here...
	log.Println("Checking machine states...")
	status := client.Status()
	status.Context = ctx
	status.Env = env
	status.Verbose = true
	if err := status.Run(); err != nil {
		return diag.FromErr(err)
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
			return diag.FromErr(err)
		}
	}

	// we're done!
	return readVagrantInfo(ctx, client, d)
}

func resourceVagrantVMDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	ctx, cancelFunc := contextWithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	defer cancelFunc()

	client, err := vagrant.NewVagrantClient(d.Get("vagrantfile_dir").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	log.Println("Destroying vagrant...")
	cmd := client.Destroy()
	cmd.Context = ctx
	cmd.Env = buildEnvironment(d.Get("env").(map[string]interface{}))
	cmd.Verbose = true

	err = cmd.Run()
	if err != nil {
		return diag.FromErr(err)
	}
	return nil
}

func resourceVagrantVMPathToVagrantfileValidate(val interface{}, key string) ([]string, []error) {
	path := filepath.Join(val.(string), "Vagrantfile")
	if _, err := os.Stat(path); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func contextWithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
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

func checkIfVMExists(ctx context.Context, client *vagrant.VagrantClient, d *schema.ResourceData) (bool, error) {
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

	return exists, nil
}

func readVagrantInfo(ctx context.Context, client *vagrant.VagrantClient, d *schema.ResourceData) diag.Diagnostics {
	env := buildEnvironment(d.Get("env").(map[string]interface{}))

	log.Println("Getting vagrant ssh-config...")
	cmd := client.SSHConfig()
	cmd.Context = ctx
	cmd.Env = env
	cmd.Verbose = true
	if err := cmd.Run(); err != nil {
		return diag.FromErr(err)
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

	ports := make([][]map[string]int, len(keys))
	if d.Get("get_ports").(bool) {
		for i, machine := range keys {
			portCmd := client.Port()
			portCmd.Context = ctx
			portCmd.Env = env
			portCmd.MachineName = machine
			portCmd.Verbose = true
			if err := portCmd.Run(); err != nil {
				return diag.FromErr(err)
			}

			ports[i] = make([]map[string]int, len(portCmd.ForwardedPorts))
			for j, p := range portCmd.ForwardedPorts {
				port := make(map[string]int, 2)
				port["guest"] = p.Guest
				port["host"] = p.Host
				ports[i][j] = port
			}
		}
	}
	d.Set("ports", ports)

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
