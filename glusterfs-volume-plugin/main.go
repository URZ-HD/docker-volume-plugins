package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"strconv"

	"github.com/docker/go-plugins-helpers/volume"
	mountedvolume "github.com/marcelo-ochoa/docker-volume-plugins/mounted-volume"
)

type gfsDriver struct {
	servers []string
	mountedvolume.Driver
}

func (p *gfsDriver) Validate(req *volume.CreateRequest) error {

	_, serversDefinedInOpts := req.Options["servers"]
	_, glusteroptsInOpts := req.Options["glusteropts"]

	if len(p.servers) > 0 && (serversDefinedInOpts || glusteroptsInOpts) {
		return fmt.Errorf("SERVERS is set, options are not allowed")
	}
	if serversDefinedInOpts && glusteroptsInOpts {
		return fmt.Errorf("servers is set, glusteropts are not allowed")
	}
	if len(p.servers) == 0 && !serversDefinedInOpts && !glusteroptsInOpts {
		return fmt.Errorf("One of SERVERS, driver_opts.servers or driver_opts.glusteropts must be specified")
	}

	return nil
}

func (p *gfsDriver) MountOptions(req *volume.CreateRequest) []string {

	servers, serversDefinedInOpts := req.Options["servers"]
	glusteropts, _ := req.Options["glusteropts"]
	enable_acl, _ := strconv.ParseBool(req.Options["acl"])

	var args []string

	if len(p.servers) > 0 {
		for _, server := range p.servers {
			args = append(args, "-s", server)
		}
		// SS: adding global option to allow posix ACL
		if enable_acl {
			args = append(args, "--acl")
		}
		args = AppendVolumeOptionsByVolumeName(args, req.Name)
	} else if serversDefinedInOpts {
		for _, server := range strings.Split(servers, ",") {
			args = append(args, "-s", server)
		}		
		// SS: adding global option to allow posix ACL		
		if enable_acl {
			args = append(args, "--acl")		
		}
		args = AppendVolumeOptionsByVolumeName(args, req.Name)
	} else {
		args = strings.Split(glusteropts, " ")
	}

	args = append(args, "--logger=syslog")

	return args
}

func (p *gfsDriver) PreMount(req *volume.MountRequest) error {
	return nil
}

func (p *gfsDriver) PostMount(req *volume.MountRequest) {
}

// AppendVolumeOptionsByVolumeName appends the command line arguments into the current argument list given the volume name
func AppendVolumeOptionsByVolumeName(args []string, volumeName string) []string {
	parts := strings.SplitN(volumeName, "/", 2)
	ret := append(args, "--volfile-id="+parts[0])
	if len(parts) == 2 {
		ret = append(ret, "--subdir-mount=/"+parts[1])
	}
	return ret
}

func buildDriver() *gfsDriver {
	var servers []string
	if os.Getenv("SERVERS") != "" {
		servers = strings.Split(os.Getenv("SERVERS"), ",")
	}
	d := &gfsDriver{
		Driver:  *mountedvolume.NewDriver("glusterfs", true, "gfs", "local"),
		servers: servers,
	}

	if os.Getenv("SECURE_MANAGEMENT") != "" {
		file, err := os.Create("/var/lib/glusterd/secure-access")
		if err != nil {
			log.Fatal("Could not create secure-access file: "+err.Error())
		}
		defer file.Close()
	}

	d.Init(d)
	return d
}

func spawnSyslog() {
	cmd := exec.Command("rsyslogd", "-n")
	cmd.Start()
}

func main() {
	spawnSyslog()
	d := buildDriver()
	defer d.Close()
	d.ServeUnix()
}
