package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"strconv"

	"github.com/manifoldco/promptui"
)

type Config struct {
	APIKey string `json:"api_key"`
	TeamID string `json:"team_id"`
	Email  string `json:"email"`
}

type Space struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Folder struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type List struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Task struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status struct {
		Status string `json:"status"`
	} `json:"status"`
	URL       string `json:"url"`
	Assignees []struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"assignees"`
}

// Load config
func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find home directory: %v", err)
	}

	configDir := filepath.Join(homeDir, ".config", "clicktime-cli")
	configFile := filepath.Join(configDir, "config.json")

	// Check for config, create dir if needed
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create config directory: %v", err)
		}
	}

	// Try to load existing config
	config := &Config{}
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err == nil {
			if err := json.Unmarshal(data, config); err == nil {
				// Config loaded successfully
				return config, nil
			}
		}
	}

	// No valid config file, prompt user for values
	fmt.Println("First time setup - please enter your ClickUp credentials")

	// Prompt for API key
	apiKeyPrompt := promptui.Prompt{
		Label: "ClickUp API Key",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("API key cannot be empty")
			}
			return nil
		},
	}
	apiKey, err := apiKeyPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("API key prompt failed: %v", err)
	}

	// Prompt for Team ID
	teamIDPrompt := promptui.Prompt{
		Label: "ClickUp Team ID",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("Team ID cannot be empty")
			}
			return nil
		},
	}
	teamID, err := teamIDPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("Team ID prompt failed: %v", err)
	}

	// Prompt for Email
	emailPrompt := promptui.Prompt{
		Label: "Your ClickUp Email",
		Validate: func(input string) error {
			if input == "" {
				return fmt.Errorf("Email cannot be empty")
			}
			return nil
		},
	}
	email, err := emailPrompt.Run()
	if err != nil {
		return nil, fmt.Errorf("Email prompt failed: %v", err)
	}

	// Create config
	config = &Config{
		APIKey: apiKey,
		TeamID: teamID,
		Email:  email,
	}

	// Save config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		return nil, fmt.Errorf("could not save config file: %v", err)
	}

	fmt.Println("Configuration saved successfully")
	return config, nil
}

// createRequest creates an HTTP request with common headers
func createRequest(method, url string, body io.Reader, apiKey string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	// Add common headers
	req.Header.Add("accept", "application/json")
	req.Header.Add("Authorization", apiKey)

	return req, nil
}

func main() {
	// Load or create configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		return
	}

	fmt.Printf("Using ClickUp API with Team ID: %s, Email: %s\n", config.TeamID, config.Email)

	// Continue with the existing flow, but use config values instead of constants
	spaces := fetchSpaces(config.APIKey, config.TeamID)

	// Space selection prompt
	spacePrompt := promptui.Select{
		Label: "Select a Space",
		Items: spaces,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "👉 {{ .Name | cyan }}",
			Inactive: "  {{ .Name }}",
			Selected: "🍅 {{ .Name | green }}",
		},
	}

	spaceIndex, _, err := spacePrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	selectedSpace := spaces[spaceIndex]
	fmt.Printf("Selected space: %s (ID: %s)\n", selectedSpace.Name, selectedSpace.ID)

	// Fetch folders for the selected space
	folders := fetchFolders(config.APIKey, selectedSpace.ID)

	// Folder selection prompt
	folderPrompt := promptui.Select{
		Label: "Select a Folder",
		Items: folders,
		HideHelp: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "👉 {{ .Name | cyan }}",
			Inactive: "  {{ .Name }}",
			Selected: "🍅 {{ .Name | green }}",
		},
	}

	folderIndex, _, err := folderPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	selectedFolder := folders[folderIndex]
	fmt.Printf("Selected folder: %s (ID: %s)\n", selectedFolder.Name, selectedFolder.ID)

	// Fetch lists for the selected folder
	lists := fetchLists(config.APIKey, selectedFolder.ID)

	// List selection prompt
	listPrompt := promptui.Select{
		Label: "Select a List",
		Items: lists,
		HideHelp: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "👉 {{ .Name | cyan }}",
			Inactive: "  {{ .Name }}",
			Selected: "🍅 {{ .Name | green }}",
		},
	}

	listIndex, _, err := listPrompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed: %v\n", err)
		return
	}

	selectedList := lists[listIndex]
	fmt.Printf("Selected list: %s (ID: %s)\n", selectedList.Name, selectedList.ID)

	// Fetch all tasks
	tasks := fetchAllTasks(config.APIKey, selectedList.ID)

	// Filter tasks by email client-side
	var myTasks []Task
	for _, task := range tasks {
		for _, assignee := range task.Assignees {
			if assignee.Email == config.Email {
				myTasks = append(myTasks, task)
				break
			}
		}
	}

	// If no tasks were found, show a message
	if len(myTasks) == 0 {
		fmt.Println("No tasks assigned to you in this list.")
		return
	}

	// Continue tracking time until user wants to exit
	for {
		// Task selection prompt
		taskPrompt := promptui.Select{
			Label: "Select a Task",
			Items: myTasks,
			HideHelp: true,
			Templates: &promptui.SelectTemplates{
				Label:    "{{ . }}",
				Active:   "👉 {{ .Name | cyan }} ({{ .Status.Status }})",
				Inactive: "  {{ .Name }} ({{ .Status.Status }})",
				Selected: "🍅 {{ .Name | green }}",
			},
		}

		taskIndex, _, err := taskPrompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed: %v\n", err)
			return
		}

		selectedTask := myTasks[taskIndex]
		fmt.Printf("\nSelected task: %s\n", selectedTask.Name)
		fmt.Printf("Status: %s\n", selectedTask.Status.Status)
		fmt.Printf("URL: %s\n", selectedTask.URL)

		// Prompt for duration in hours
		durationPrompt := promptui.Prompt{
			Label:    "Duration (in hours)",
			Default:  "1",
			Validate: func(input string) error {
				_, err := strconv.ParseFloat(input, 64)
				if err != nil {
					return fmt.Errorf("Please enter a valid number")
				}
				return nil
			},
		}

		durationStr, err := durationPrompt.Run()
		if err != nil {
			fmt.Printf("Prompt failed: %v\n", err)
			return
		}

		// Convert duration to milliseconds
		durationHours, _ := strconv.ParseFloat(durationStr, 64)
		durationMs := int64(durationHours * 60 * 60 * 1000)

		// Get current time in milliseconds
		startTime := time.Now().UnixNano() / int64(time.Millisecond)

		// Create time entry
		err = createTimeEntry(config.APIKey, config.TeamID, selectedTask.ID, startTime, durationMs)
		if err != nil {
			fmt.Printf("Error creating time entry: %v\n", err)
			return
		}

		fmt.Printf("\nTime entry created successfully!\n")
		fmt.Printf("Duration: %.1f hours\n", durationHours)

		// Ask if user wants to continue tracking time
		continuePrompt := promptui.Prompt{
			Label:     "Track time for another task",
			IsConfirm: true,
		}

		continueResult, err := continuePrompt.Run()
		if err != nil || strings.ToLower(continueResult) != "y" {
			fmt.Println("Exiting. Have a productive day!")
			break
		}
	}
}

func fetchSpaces(apiKey, teamID string) []Space {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/space", teamID)

	req, err := createRequest("GET", url, nil, apiKey)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response struct {
		Spaces []Space `json:"spaces"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil
	}

	return response.Spaces
}

func fetchFolders(apiKey, spaceID string) []Folder {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/space/%s/folder", spaceID)

	req, err := createRequest("GET", url, nil, apiKey)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response struct {
		Folders []Folder `json:"folders"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil
	}

	return response.Folders
}

func fetchLists(apiKey, folderID string) []List {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/folder/%s/list", folderID)

	req, err := createRequest("GET", url, nil, apiKey)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response struct {
		Lists []List `json:"lists"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil
	}

	return response.Lists
}

func fetchAllTasks(apiKey, listID string) []Task {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/task", listID)

	req, err := createRequest("GET", url, nil, apiKey)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response struct {
		Tasks []Task `json:"tasks"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil
	}

	return response.Tasks
}

func createTimeEntry(apiKey, teamID, taskID string, startTime int64, durationMs int64) error {
	url := fmt.Sprintf("https://api.clickup.com/api/v2/team/%s/time_entries", teamID)

	// Create the payload
	payload := fmt.Sprintf(`{
		"start": %d,
		"duration": %d,
		"tid": "%s"
	}`, startTime, durationMs, taskID)

	// Create request
	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", apiKey)

	// Make request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// Check response
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("API error: %s - %s", res.Status, string(body))
	}

	return nil
}
