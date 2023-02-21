package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli"
	"gopkg.in/yaml.v2"
)

type Hosts struct {
	All struct {
		Hosts map[string]HostDetails `yaml:"hosts"`
	} `yaml:"all"`
}

type HostDetails struct {
	AnsibleHost       string `yaml:"ansible_host"`
	NodeSeed          string `yaml:"node_seed"`
	NodePubkey        string `yaml:"node_pubkey"`
	IsHostedOnAzure   bool   `yaml:"is_hosted_on_azure"`
	AnsibleUser       string `yaml:"ansible_user"`
	PostgresPath      string `yaml:"postgres_path"`
	PostgresMountPath string `yaml:"postgres_mount_path"`
}

type FirewallChecklist struct {
	Firewall struct {
		ValidatorNodes map[string]ValidatorNode `yaml:"validator-nodes"`
	} `yaml:"firewall-checklist"`
}

type ValidatorNode struct {
	Ports       []string `yaml:"ports"`
	IPWhitelist []string `yaml:"ip-whitelist"`
}

func checkHostFile(hostsPath, outputPath string) {
	//hostsPath := flag.String("hosts", "", "path to hosts file")
	//outputPath := flag.String("output", "", "path to output file")
	//flag.Parse()

	if hostsPath == "" {
		log.Fatal("hosts file path not specified")
	}

	data, err := os.ReadFile(hostsPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Checking hosts file: ", hostsPath)

	hosts := Hosts{}
	err = yaml.Unmarshal(data, &hosts)
	if err != nil {
		log.Fatal(err)
	}

	firewallChecklist := FirewallChecklist{
		Firewall: struct {
			ValidatorNodes map[string]ValidatorNode `yaml:"validator-nodes"`
		}{ValidatorNodes: map[string]ValidatorNode{}},
	}

	for hostName, hostDetails := range hosts.All.Hosts {
		if hostDetails.AnsibleHost != "" {
			validatorNode := ValidatorNode{
				Ports: []string{
					fmt.Sprintf("%v # %v", 2210, "archives"),
					fmt.Sprintf("%v # %v", 11625, "blockchain gossip protocol"),
				},
				IPWhitelist: []string{},
			}

			for otherHostName, otherHostDetails := range hosts.All.Hosts {
				if otherHostName != hostName && otherHostDetails.AnsibleHost != "" {
					validatorNode.IPWhitelist = append(validatorNode.IPWhitelist, fmt.Sprintf("%v # %v", strings.ReplaceAll(otherHostDetails.AnsibleHost, "'", ""), otherHostName))
				}
			}

			firewallChecklist.Firewall.ValidatorNodes[hostDetails.AnsibleHost] = validatorNode
			// append the ansible hostnames to the validator nodes

		}
	}

	outputData, err := yaml.Marshal(firewallChecklist)
	if err != nil {
		log.Fatal(err)
	}

	outputData = bytes.Replace(outputData, []byte("'"), []byte(""), -1)

	if outputPath == "" {
		fmt.Println(string(outputData))
	} else {
		outputFile, err := os.Create(outputPath)
		if err != nil {
			log.Fatal(err)
		}
		defer outputFile.Close()

		_, err = outputFile.Write(outputData)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Firewall checklist file created at %s\n\n", outputPath)

	}
}

func main() {
	fmt.Println("Running firewall-checklist")
	app := cli.NewApp()
	app.Name = "firewall-checklist"
	app.Usage = "Creates a firewall checklist for the hosts in the hosts folder"
	app.Version = "1.0.0"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "path, p",
			Value: ".",
			Usage: "Path to the ansible directory",
		},
	}

	app.Action = func(c *cli.Context) error {
		// check if the hosts folder exists
		hostsPath := filepath.Join(c.String("path"), "hosts")
		if _, err := os.Stat(hostsPath); os.IsNotExist(err) {
			log.Fatal("hosts folder does not exist")
		}

		// check if the firewall-checklist folder exists
		outputPath := filepath.Join(c.String("path"), "firewall-checklist")
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			// create the firewall-checklist folder
			err = os.Mkdir(outputPath, 0755)
			if err != nil {
				log.Fatal(err)
			}
		}

		hostsFiles, err := os.ReadDir(hostsPath)
		if err != nil {
			log.Fatal(err)
		}

		for _, hostsFile := range hostsFiles {
			if hostsFile.IsDir() {
				continue
			}
			hostsFilePath := filepath.Join(hostsPath, hostsFile.Name())
			outputFilePath := filepath.Join(outputPath, hostsFile.Name())

			checkHostFile(hostsFilePath, outputFilePath)
		}

		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
