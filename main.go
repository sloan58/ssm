package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

type Connection struct {
	Name     string `json:"name"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	SSHKey   string `json:"ssh_key"`
}

type Defaults struct {
	Port     int    `json:"port"`
	Username string `json:"username"`
	SSHKey   string `json:"ssh_key"`
}

var (
	connectionsFile string
	defaultsFile    string
)

// Print the colorful heading
func printHeading() {
	// Define the colors
	titleColor := color.New(color.FgHiCyan).Add(color.Bold)
	borderColor := color.New(color.FgMagenta).Add(color.Bold)

	// Print the heading with colors
	borderColor.Println("#################")
	borderColor.Print("#  ")
	titleColor.Print("SSH Manager")
	borderColor.Println("  #")
	borderColor.Println("#################")
}

// Main menu function
func main() {
	setupConfigPaths()

	printHeading() // Call the function to print the heading

	for {
		printMainMenu()
		choice := readUserInput()
		switch choice {
		case "1":
			listConnections()
		case "2":
			addConnection()
		case "3":
			deleteConnection()
		case "4":
			editDefaultSettings()
		case "5":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid choice. Please try again.")
		}
	}
}

// Set up configuration paths
func setupConfigPaths() {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("Error getting current user:", err)
		return
	}

	configDir := filepath.Join(usr.HomeDir, ".config", "ssm")
	for _, path := range []string{configDir, filepath.Join(configDir, "connections.json"), filepath.Join(configDir, "defaults.json")} {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			err = os.MkdirAll(configDir, os.ModePerm)
			if err != nil {
				fmt.Println("Error creating config directories:", err)
			}
			if filepath.Base(path) != "ssm" {
				file, err := os.Create(path)
				defer file.Close()
				if err != nil {
					fmt.Println("Error creating config file:", err)
				}
			}
		}
	}

	connectionsFile = filepath.Join(configDir, "connections.json")
	defaultsFile = filepath.Join(configDir, "defaults.json")
}

// Read user input
func readUserInput() string {
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return input[:len(input)-1]
}

// Print main menu
func printMainMenu() {
	fmt.Println(`
1. List connections and connect
2. Add connection
3. Delete connection
4. Edit default settings
5. Exit
Please enter your choice:`)
}

// Load connections from file
func loadConnections() ([]Connection, error) {
	// Check if file exists
	if _, err := os.Stat(connectionsFile); os.IsNotExist(err) {
		// Create empty file with an empty array if it doesn't exist
		err = ioutil.WriteFile(connectionsFile, []byte("[]"), 0644)
		if err != nil {
			return nil, err
		}
	}

	file, err := os.Open(connectionsFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var connections []Connection
	err = json.NewDecoder(file).Decode(&connections)
	if err != nil {
		// Handle empty or improperly formatted file
		if err == io.EOF {
			connections = []Connection{}
		} else {
			return nil, err
		}
	}
	return connections, nil
}

// Load default settings from file
func loadDefaults() (Defaults, error) {
	var defaults Defaults

	// Check if file exists
	if _, err := os.Stat(defaultsFile); os.IsNotExist(err) {
		// Initialize defaults and save them to file
		defaults = Defaults{Port: 22, Username: "root", SSHKey: filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")}
		defaultsJson, err := json.Marshal(defaults)
		if err != nil {
			return defaults, err
		}
		err = ioutil.WriteFile(defaultsFile, defaultsJson, 0644)
		if err != nil {
			return defaults, err
		}
		return defaults, nil
	}

	file, err := os.Open(defaultsFile)
	if err != nil {
		return Defaults{}, err
	}
	defer file.Close()

	err = json.NewDecoder(file).Decode(&defaults)
	if err != nil {
		// Handle empty or improperly formatted file
		if err == io.EOF {
			defaults = Defaults{Port: 22, Username: "root", SSHKey: filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa")}
			defaultsJson, err := json.Marshal(defaults)
			if err != nil {
				return defaults, err
			}
			err = ioutil.WriteFile(defaultsFile, defaultsJson, 0644)
			if err != nil {
				return defaults, err
			}
			return defaults, nil
		} else {
			return Defaults{}, err
		}
	}
	return defaults, nil
}

// Save default settings to file
func saveDefaults(defaults Defaults) error {
	file, err := os.Create(defaultsFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(defaults)
}

// List connections and prompt to connect
func listConnections() {
	connections, err := loadConnections()
	if err != nil {
		fmt.Println("Error loading connections:", err)
		return
	}

	if len(connections) == 0 {
		fmt.Println("No connections found.")
		return
	}

	// Create a new table writer
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"#", "Name", "Host", "Port", "Username", "SSH Key"})

	// Define color functions
	indexColor := color.New(color.FgCyan).SprintFunc()
	nameColor := color.New(color.FgGreen).SprintFunc()
	hostColor := color.New(color.FgMagenta).SprintFunc()
	portColor := color.New(color.FgYellow).SprintFunc()
	usernameColor := color.New(color.FgBlue).SprintFunc()
	sshKeyColor := color.New(color.FgRed).SprintFunc()

	// Populate rows with colorized data
	for i, conn := range connections {
		table.Append([]string{
			indexColor(strconv.Itoa(i + 1)),
			nameColor(conn.Name),
			hostColor(conn.Host),
			portColor(strconv.Itoa(conn.Port)),
			usernameColor(conn.Username),
			sshKeyColor(conn.SSHKey),
		})
	}

	// Set table alignment and other properties
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(true)
	table.Render()

	// Prompt for connection selection
	fmt.Print("Enter the number of the connection to connect, or 'b' to go back: ")
	input := readUserInput()
	if input == "b" {
		return
	}

	index, err := strconv.Atoi(input)
	if err != nil || index < 1 || index > len(connections) {
		fmt.Println("Invalid selection.")
		return
	}

	connection := connections[index-1]
	executeSSHCommand(connection)
}

// Add a new connection
func addConnection() {
	defaults, err := loadDefaults()
	if err != nil {
		fmt.Println("Error loading default settings:", err)
		return
	}

	// Prompt user for each field
	fmt.Println("Adding a new connection. Press Enter to use default value where applicable.")

	fmt.Printf("Name: ")
	name := readUserInput()

	fmt.Printf("Host: ")
	host := readUserInput()

	fmt.Printf("Port (default: %d): ", defaults.Port)
	portStr := readUserInput()
	port, err := strconv.Atoi(portStr)
	if err != nil {
		port = defaults.Port
	}

	fmt.Printf("Username (default: %s): ", defaults.Username)
	username := readUserInput()
	if username == "" {
		username = defaults.Username
	}

	fmt.Printf("SSH Key (default: %s): ", defaults.SSHKey)
	sshKey := readUserInput()
	if sshKey == "" {
		sshKey = defaults.SSHKey
	}

	// Create new connection
	newConnection := Connection{
		Name:     name,
		Host:     host,
		Port:     port,
		Username: username,
		SSHKey:   sshKey,
	}

	// Load existing connections
	connections, err := loadConnections()
	if err != nil {
		fmt.Println("Error loading connections:", err)
		return
	}

	// Add new connection
	connections = append(connections, newConnection)

	// Save connections back to file
	file, err := os.Create(connectionsFile)
	if err != nil {
		fmt.Println("Error saving connections:", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(connections)
	if err != nil {
		fmt.Println("Error encoding connections:", err)
		return
	}

	fmt.Println("Connection added successfully!")
}

// Delete a connection
func deleteConnection() {
	connections, err := loadConnections()
	if err != nil {
		fmt.Println("Error loading connections:", err)
		return
	}

	if len(connections) == 0 {
		fmt.Println("No connections to delete.")
		return
	}

	listConnections()

	fmt.Print("Enter the number of the connection you want to delete: ")
	indexStr := readUserInput()
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 1 || index > len(connections) {
		fmt.Println("Invalid number.")
		return
	}

	// Remove the selected connection
	connections = append(connections[:index-1], connections[index:]...)

	// Save the updated connections back to the file
	file, err := os.Create(connectionsFile)
	if err != nil {
		fmt.Println("Error saving connections:", err)
		return
	}
	defer file.Close()

	err = json.NewEncoder(file).Encode(connections)
	if err != nil {
		fmt.Println("Error encoding connections:", err)
		return
	}

	fmt.Println("Connection deleted successfully!")
}

// Edit default settings
func editDefaultSettings() {
	defaults, err := loadDefaults()
	if err != nil {
		fmt.Println("Error loading default settings:", err)
		return
	}

	fmt.Println("Editing default settings. Press Enter to keep the current value where applicable.")

	fmt.Printf("Port (current: %d): ", defaults.Port)
	portStr := readUserInput()
	if portStr != "" {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Println("Invalid port number. Keeping current value.")
		} else {
			defaults.Port = port
		}
	}

	fmt.Printf("Username (current: %s): ", defaults.Username)
	username := readUserInput()
	if username != "" {
		defaults.Username = username
	}

	fmt.Printf("SSH Key (current: %s): ", defaults.SSHKey)
	sshKey := readUserInput()
	if sshKey != "" {
		defaults.SSHKey = sshKey
	}

	err = saveDefaults(defaults)
	if err != nil {
		fmt.Println("Error saving default settings:", err)
	} else {
		fmt.Println("Default settings updated successfully!")
	}
}

// Helper function to execute the SSH command based on the connection
func executeSSHCommand(connection Connection) {
	sshPath := "ssh" // Default for Unix-like systems

	if runtime.GOOS == "windows" {
		// Check if ssh.exe is in SYSTEM32
		if path, err := exec.LookPath("ssh.exe"); err == nil {
			sshPath = path
		} else {
			// If not found, you can handle installing or guiding the user to install OpenSSH
			log.Fatal("ssh.exe not found in PATH. Please install OpenSSH client.")
			return
		}
	}

	cmd := exec.Command(sshPath, fmt.Sprintf("%s@%s", connection.Username, connection.Host), "-p", strconv.Itoa(connection.Port), "-i", connection.SSHKey)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to run SSH command: %v", err)
	}
}
