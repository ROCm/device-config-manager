package utils

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
	log_e "github.com/sirupsen/logrus"
)

const serviceDivider = "^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^"

type ServicePreState struct {
	Name      string // e.g. "amd-metrics-exporter.service"
	State     string // e.g. "active", "inactive", "failed", "not-loaded"
	Timestamp time.Time
	Comment   string
}

var PreStateDB = make(map[string]ServicePreState)

// connect to the system D-Bus
func getSystemdConn() (*dbus.Conn, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to system bus: %v", err)
	}
	return conn, nil
}

// service control based on action (StartUnit or StopUnit)
func controlService(action, serviceName string) error {
	conn, err := getSystemdConn()
	if err != nil {
		return err
	}
	obj := conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	call := obj.Call("org.freedesktop.systemd1.Manager."+action+"Unit", 0, serviceName, "replace")

	if call.Err != nil {
		return fmt.Errorf("D-Bus call failed: %v", call.Err)
	}
	log.Printf("Service '%s' %s triggered.\n", serviceName, strings.ToLower(action))
	return nil
}

func StartService(name string) error {
	if UnitExists(name) {
		log.Printf("Service %v already exists. Skipping restart!!", name)
		err := errors.New("service already exists")
		return err
	}
	return controlService("Start", name)
}

func StopService(name string) error {
	if !UnitExists(name) {
		log.Printf("Service %v does not exist. Skipping!!", name)
		err := errors.New("service does not exist")
		return err
	}
	return controlService("Stop", name)
}

func CleanupPreState() {
	log.Println("Cleaning up PreStateDB...")
	PreStateDB = make(map[string]ServicePreState)
	if len(PreStateDB) == 0 {
		log.Println("PreStateDB has been successfully emptied.")
	} else {
		log.Printf("Warning: PreStateDB still has %d entries.", len(PreStateDB))
	}
}

func StartServiceHandler(services []string) {
	log.Println(serviceDivider)
	log.Printf("ServicesList %v", services)
	for _, svc := range services {
		if !strings.HasSuffix(svc, ".service") {
			svc += ".service"
		}
		preState := PreStateDB[svc]
		if preState.State == "active" {
			log.Printf("Service %s prestate is active (status: %s), attempting restart", svc, preState.State)
		} else {
			log.Printf("Restarting service skipped for: %s (was %s at %s)\n", svc, preState.State, preState.Timestamp)
			continue
		}
		log.Printf("Restarting service: %s", svc)
		if err := StartService(svc); err != nil {
			log.Printf("Warning: Failed to start service %s: %v\n", svc, err)
		} else {
			log.Printf("Validating %v service status", svc)
			if CheckUnitStatusHandler(svc, "active") {
				log.Printf("Service %s (status: %s), successfully restarted", svc, "active")
			}
		}
	}
	CleanupPreState()
	log.Println(serviceDivider)
}

func StopServiceHandler(services []string) {
	log.Println(serviceDivider)
	log.Printf("ServicesList %v", services)
	for _, svc := range services {
		if !strings.HasSuffix(svc, ".service") {
			svc += ".service"
		}
		status := CheckUnitStatus(svc)
		PreStateDB[svc] = ServicePreState{
			Name:      svc,
			State:     status,
			Timestamp: time.Now(),
			Comment:   fmt.Sprintf("Service was %s before StopService", status),
		}

		preState := PreStateDB[svc]
		if preState.State != "active" {
			log.Printf("Service %s is not active (status: %s), skipping stop", svc, status)
			continue
		} else {
			log.Printf("Service %s current state is active (status: %s), attempting stop", svc, preState.State)
		}
		log.Printf("Stopping service: %s", svc)
		if err := StopService(svc); err != nil {
			log.Printf("Warning: Failed to stop service %s: %v\n", svc, err)
		} else {
			log.Printf("Validating %v service status", svc)
			if CheckUnitStatusHandler(svc, "not-loaded") {
				log.Printf("Service %s (status: %s), successfully stopped", svc, "not-loaded")
			}
		}
	}
	log.Println(serviceDivider)
}

// checking if a systemd unit exists
func UnitExists(unitName string) bool {
	// sleep for 2 seconds to determine the status
	time.Sleep(2 * time.Second)
	log.Printf("Checking if %v exists", unitName)
	conn, err := dbus.SystemBus()
	if err != nil {
		return false
	}

	systemd := conn.Object("org.freedesktop.systemd1", "/org/freedesktop/systemd1")
	var unitPath dbus.ObjectPath

	err = systemd.Call("org.freedesktop.systemd1.Manager.GetUnit", 0, unitName).Store(&unitPath)
	return err == nil
}

// check service status
func CheckUnitStatus(name string) string {
	// sleep for 10 seconds to determine the status
	time.Sleep(10 * time.Second)
	conn, err := getSystemdConn()
	if err != nil {
		log_e.Errorf("err: %+v", err)
		return ""
	}
	manager := conn.Object("org.freedesktop.systemd1", dbus.ObjectPath("/org/freedesktop/systemd1"))
	var unitPath dbus.ObjectPath

	err = manager.Call("org.freedesktop.systemd1.Manager.GetUnit", 0, name).Store(&unitPath)
	if err != nil {
		if dbusErr, ok := err.(dbus.Error); ok {
			if dbusErr.Name == "org.freedesktop.systemd1.NoSuchUnit" {
				return "not-loaded"
			}
		}
		log_e.Errorf("failed to get unit: %v", err)
		return ""
	}

	unit := conn.Object("org.freedesktop.systemd1", unitPath)
	variant, err := unit.GetProperty("org.freedesktop.systemd1.Unit.ActiveState")
	if err != nil {
		log_e.Errorf("failed to get ActiveState: %v", err)
		return ""
	}

	activeState, ok := variant.Value().(string)
	if !ok {
		log_e.Errorf("unexpected type for ActiveState")
		return ""
	}

	return activeState
}

func CheckUnitStatusHandler(svc string, exp_status string) bool {
	status := CheckUnitStatus(svc)
	if status != exp_status {
		log.Printf("Service %s (status: %s), (expected status: %v)", svc, status, exp_status)
		return false
	}
	return true
}
