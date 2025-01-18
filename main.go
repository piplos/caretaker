package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func main() {
	trackedContainer := os.Getenv("TRACKED_CONTAINER")
	restartedContainer := os.Getenv("RESTARTED_CONTAINER")
	sleepInterval, _ := strconv.Atoi(os.Getenv("SLEEP_INTERVAL"))
	cpuThreshold, _ := strconv.ParseFloat(os.Getenv("CPU_THRESHOLD"), 64)

	if trackedContainer == "" || restartedContainer == "" {
		fmt.Println("Error: TRACKED_CONTAINER and RESTARTED_CONTAINER environment variables are required")
		os.Exit(1)
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	for {
		trackedContainerID, err := getContainerID(cli, trackedContainer)
		if err != nil {
			fmt.Println("Error getting tracked container ID:", err)
			time.Sleep(time.Duration(sleepInterval) * time.Second)
			continue
		}

		cpuUsage, err := getCPUUsage(cli, trackedContainerID)
		if err != nil {
			fmt.Println("Error getting CPU usage:", err)
			time.Sleep(time.Duration(sleepInterval) * time.Second)
			continue
		}

		if cpuUsage > cpuThreshold {
			restartedContainerID, err := getContainerID(cli, restartedContainer)
			if err != nil {
				fmt.Println("Error getting restarted container ID:", err)
				time.Sleep(time.Duration(sleepInterval) * time.Second)
				continue
			}

			err = cli.ContainerRestart(context.Background(), restartedContainerID, container.StopOptions{})
			if err != nil {
				fmt.Println("Error restarting restarted container:", err)
			}
		}

		time.Sleep(time.Duration(sleepInterval) * time.Second)
	}
}

func getContainerID(cli *client.Client, containerName string) (string, error) {
	containers, err := cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, container := range containers {
		for _, name := range container.Names {
			if strings.Contains(name, containerName) {
				return container.ID, nil
			}
		}
	}

	return "", fmt.Errorf("container %s not found", containerName)
}

func getCPUUsage(cli *client.Client, containerID string) (float64, error) {
	stats, err := cli.ContainerStatsOneShot(context.Background(), containerID)
	if err != nil {
		return 0, err
	}
	defer stats.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return 0, err
	}

	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage - v.PreCPUStats.SystemUsage)
	cpuUsage := (cpuDelta / systemDelta) * float64(v.CPUStats.OnlineCPUs) * 100.0

	return cpuUsage, nil
}
