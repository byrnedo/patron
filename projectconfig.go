package main

import (
	"fmt"
	"github.com/byrnedo/capitan/container"
	"github.com/byrnedo/capitan/helpers"
	. "github.com/byrnedo/capitan/logger"
	"github.com/codeskyblue/go-sh"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
)

var (
	allDone = make(chan bool, 1)
)

type ProjectConfig struct {
	ProjectName           string
	ProjectSeparator      string
	ContainerSettingsList SettingsList
}

type SettingsList []container.Container

func (s SettingsList) Len() int {
	return len(s)
}
func (s SettingsList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s SettingsList) Less(i, j int) bool {
	return s[i].Placement < s[j].Placement
}

func (settings *ProjectConfig) LaunchCleanupWatcher() {

	var (
		killBegan     = make(chan bool, 1)
		killDone      = make(chan bool, 1)
		stopDone      = make(chan bool, 1)
		signalChannel = make(chan os.Signal)
	)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	go func() {

		var (
			killing bool
		)

		for {
			select {
			case <-killBegan:
				killing = true
			case <-stopDone:
				if !killing {
					allDone <- true
				}
			case <-killDone:
				allDone <- true
			}
		}
	}()

	go func() {
		var calls int
		for {
			sig := <-signalChannel
			switch sig {
			case os.Interrupt, syscall.SIGTERM:
				calls++
				if calls == 1 {
					go func() {
						settings.CapitanStop(nil, false)
						stopDone <- true
					}()
				} else if calls == 2 {
					killBegan <- true
					settings.CapitanKill(nil, false)
					killDone <- true
				}
			default:
				Debug.Println("Unhandled signal", sig)
			}
		}
		Info.Println("Done cleaning up")
	}()
}

func newerImage(container string, image string) bool {

	conImage := helpers.GetContainerImageId(container)
	localImage := helpers.GetImageId(image)
	if conImage != "" && localImage != "" && conImage != localImage {
		return true
	}
	return false
}

func haveArgsChanged(container string, runArgs []interface{}) bool {

	uniqueLabel := fmt.Sprintf("%s", runArgs)
	if helpers.GetContainerUniqueLabel(container) != uniqueLabel {
		return true
	}
	return false
	// remove and restart

}

func (settings *ProjectConfig) CapitanCreate(dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)

	for _, set := range settings.ContainerSettingsList {

		if helpers.GetImageId(set.Image) == "" {
			Warning.Printf("Capitan was unable to find image %s locally\n", set.Image)

			if set.Build != "" {
				Info.Println("Building image")
				if err := set.BuildImage(); err != nil {
					return err
				}
			} else {
				Info.Println("Pulling image")
				if err := helpers.PullImage(set.Image); err != nil {
					return err
				}
			}
		}

		if err := set.Create(dryRun); err != nil {
			return err
		}
	}
	return nil

}

// The 'up' command
//
// Creates a container if it doesn't exist
// Starts a container if stopped
// Recreates a container if the container's image has a newer id locally
// OR if the command used to create the container is now changed (i.e.
// config has changed.
func (settings *ProjectConfig) CapitanUp(attach bool, dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)

	wg := sync.WaitGroup{}

	for _, set := range settings.ContainerSettingsList {
		var (
			err error
		)

		if helpers.GetImageId(set.Image) == "" {
			Warning.Printf("Capitan was unable to find image %s locally\n", set.Image)

			if set.Build != "" {
				Info.Println("Building image")
				if err := set.BuildImage(); err != nil {
					return err
				}
			} else {
				Info.Println("Pulling image")
				if err := helpers.PullImage(set.Image); err != nil {
					return err
				}
			}
		}

		//create new
		if !helpers.ContainerExists(set.Name) {
			if err = set.Run(attach, dryRun, &wg); err != nil {
				return err
			}
			continue
		}


		if newerImage(set.Name, set.Image) {
			// remove and restart
			Info.Println("Removing (different image available):", set.Name)
			if err = set.RecreateAndRun(attach, dryRun, &wg); err != nil {
				return err
			}

			continue
		}

		if haveArgsChanged(set.Name, set.GetRunArguments()) {
			// remove and restart
			Info.Println("Removing (run arguments changed):", set.Name)
			if err = set.RecreateAndRun(attach, dryRun, &wg); err != nil {
				return err
			}
			continue
		}

		//attach if running
		if helpers.ContainerIsRunning(set.Name) {
			Info.Println("Already running " + set.Name)
			if attach {
				Info.Println("Attaching")
				if err := set.Attach(&wg); err != nil {
					return err
				}
			}
			continue
		}

		Info.Println("Starting " + set.Name)

		if dryRun {
			continue
		}

		//start if stopped
		if err = set.Start(attach, &wg); err != nil {
			return err
		}
		continue

	}
	wg.Wait()
	if !dryRun && attach {
		<-allDone
	}
	return nil
}

// Starts stopped containers
func (settings *ProjectConfig) CapitanStart(attach bool, dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)
	wg := sync.WaitGroup{}
	for _, set := range settings.ContainerSettingsList {
		if helpers.ContainerIsRunning(set.Name) {
			Info.Println("Already running " + set.Name)
			if attach {
				Info.Println("Attaching")
				if err := set.Attach(&wg); err != nil {
					return err
				}
			}
			continue
		}
		Info.Println("Starting " + set.Name)
		if !dryRun {
			if err := set.Start(attach, &wg); err != nil {
				return err
			}
		}
	}
	wg.Wait()
	if !dryRun && attach {
		<-allDone
	}
	return nil
}

// Command to restart all containers
func (settings *ProjectConfig) CapitanRestart(args []string, dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)
	for _, set := range settings.ContainerSettingsList {
		Info.Println("Restarting " + set.Name)
		if !dryRun {
			if err := set.Restart(args); err != nil {
				return err
			}
		}
	}
	return nil
}

// Print all container IPs
func (settings *ProjectConfig) CapitanIP() error {
	sort.Sort(settings.ContainerSettingsList)
	for _, set := range settings.ContainerSettingsList {
		ip := set.IP()
		Info.Printf("%s: %s", set.Name, ip)
	}
	return nil
}

// Stream all container logs
func (settings *ProjectConfig) CapitanLogs() error {
	sort.Sort(settings.ContainerSettingsList)
	var wg sync.WaitGroup
	for _, set := range settings.ContainerSettingsList {
		var (
			ses *sh.Session
			err error
		)
		if ses, err = set.Logs(); err != nil {
			Error.Println("Error getting log for " + set.Name + ": " + err.Error())
			continue
		}

		wg.Add(1)

		go func() {
			ses.Wait()
			wg.Done()
		}()

	}
	wg.Wait()
	return nil
}

// Stream all container stats
func (settings *ProjectConfig) CapitanStats() error {
	var (
		args []interface{}
	)
	sort.Sort(settings.ContainerSettingsList)

	args = make([]interface{}, len(settings.ContainerSettingsList))

	for i, set := range settings.ContainerSettingsList {
		args[i] = set.Name
	}

	ses := sh.NewSession()
	ses.Command("docker", append([]interface{}{"stats"}, args...)...)
	ses.Start()
	ses.Wait()
	return nil
}

// Print `docker ps` ouptut for all containers in project
func (settings *ProjectConfig) CapitanPs(args []string) error {
	sort.Sort(settings.ContainerSettingsList)
	allArgs := append([]interface{}{"ps"}, helpers.ToInterfaceSlice(args)...)
	for _, set := range settings.ContainerSettingsList {
		allArgs = append(allArgs, "-f", "name="+set.Name)
	}
	var (
		err error
		out []byte
	)
	if out, err = helpers.RunCmd(allArgs...); err != nil {
		return err
	}
	Info.Print(string(out))
	return nil
}

// Kill all running containers in project
func (settings *ProjectConfig) CapitanKill(args []string, dryRun bool) error {
	sort.Sort(sort.Reverse(settings.ContainerSettingsList))
	for _, set := range settings.ContainerSettingsList {
		if !helpers.ContainerIsRunning(set.Name) {
			Info.Println("Already dead:", set.Name)
			continue
		}
		Info.Println("Killing " + set.Name)
		if !dryRun {
			if err := set.Kill(args); err != nil {
				return err
			}
		}
	}
	return nil
}

// Stops the containers in the project
func (settings *ProjectConfig) CapitanStop(args []string, dryRun bool) error {
	sort.Sort(sort.Reverse(settings.ContainerSettingsList))
	for _, set := range settings.ContainerSettingsList {
		if !helpers.ContainerIsRunning(set.Name) {
			Info.Println("Already dead:", set.Name)
			continue
		}
		Info.Println("Stopping " + set.Name)
		if !dryRun {
			if err := set.Stop(args); err != nil {
				return err
			}
		}
	}
	return nil
}

// Remove all containers in project
func (settings *ProjectConfig) CapitanRm(args []string, dryRun bool) error {
	sort.Sort(sort.Reverse(settings.ContainerSettingsList))
	for _, set := range settings.ContainerSettingsList {

		if !dryRun && helpers.ContainerExists(set.Name) {
			Info.Println("Removing " + set.Name)
			if err := set.Rm(args); err != nil {
				return err
			}
		} else {
			Info.Println("Container doesn't exist:", set.Name)
		}
	}
	return nil
}

// The build command
func (settings *ProjectConfig) CapitanBuild(dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)
	for _, set := range settings.ContainerSettingsList {
		if len(set.Build) == 0 {
			continue
		}
		Info.Println("Building " + set.Name)
		if !dryRun {
			if err := set.BuildImage(); err != nil {
				return err
			}
		}

	}
	return nil
}

// The build command
func (settings *ProjectConfig) CapitanPull(dryRun bool) error {
	sort.Sort(settings.ContainerSettingsList)
	for _, set := range settings.ContainerSettingsList {
		if len(set.Build) > 0 || set.Image == "" {
			continue
		}
		Info.Println("Pulling", set.Image, "for", set.Name)
		if !dryRun {
			if err := helpers.PullImage(set.Image); err != nil {
				return err
			}
		}

	}
	return nil
}