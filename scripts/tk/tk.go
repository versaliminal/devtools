package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const dateFormat = "2006-01-02"

type Task struct {
	ID            int
	Name          string
	CreatedDate   string
	Status        string
	CompletedDate string
	Subtasks      []Subtask
	LineNumber    int
}

type Subtask struct {
	ID            int
	Description   string
	CreatedDate   string
	CompletedDate string
	Status        string
	LineNumber    int
}

const (
	dimColor    = lipgloss.Color("241")
	openColor   = lipgloss.Color("#cfae23")
	closedColor = lipgloss.Color("#665c71")
	taskColor   = lipgloss.Color("#d3a8d3")
)

var (
	dimStyle = lipgloss.NewStyle().
			Foreground(dimColor)
	taskStyle = lipgloss.NewStyle().
			Foreground(taskColor)
	nameStyle = lipgloss.NewStyle().
			Foreground(taskColor)
	openStyle = lipgloss.NewStyle().
			Foreground(openColor)
	closedStyle = lipgloss.NewStyle().
			Foreground(closedColor)
)

var (
	openIcon   = openStyle.Render("[ ]")
	closedIcon = openStyle.Render("[") + closedStyle.Render("x") + openStyle.Render("]")
)

func getVimwikiHome() (string, error) {
	home := os.Getenv("VIMWIKI_HOME")
	if home == "" {
		return "", fmt.Errorf("VIMWIKI_HOME environment variable is not set")
	}
	return home, nil
}

func getTasksDir() (string, error) {
	home, err := getVimwikiHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "tasks"), nil
}

func getIndexPath() (string, error) {
	tasksDir, err := getTasksDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tasksDir, "index.md"), nil
}

func getProjectPath(projectName string) (string, error) {
	tasksDir, err := getTasksDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tasksDir, projectName+".md"), nil
}

func ensureTasksDir() error {
	tasksDir, err := getTasksDir()
	if err != nil {
		return err
	}
	return os.MkdirAll(tasksDir, 0755)
}

func ensureIndexFile() error {
	indexPath, err := getIndexPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		file, err := os.Create(indexPath)
		if err != nil {
			return err
		}
		defer file.Close()
		fmt.Fprintf(file, "# Index\n\n")
	}
	return nil
}

func ensureProjectFile(projectName string) error {
	projectPath, err := getProjectPath(projectName)
	if err != nil {
		return err
	}

	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		file, err := os.Create(projectPath)
		if err != nil {
			return err
		}
		defer file.Close()
		fmt.Fprintf(file, "# %s\n\n", projectName)
	}
	return nil
}

func ensureLinkInIndex(projectName string) error {
	indexPath, err := getIndexPath()
	if err != nil {
		return err
	}

	file, err := os.Open(indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	linkPattern := regexp.MustCompile(fmt.Sprintf(`^\* \[\[%s\.md\|%s\]\]`, regexp.QuoteMeta(projectName), regexp.QuoteMeta(projectName)))
	for scanner.Scan() {
		if linkPattern.MatchString(scanner.Text()) {
			return nil
		}
	}
	file.Close()

	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	fmt.Fprintf(f, "* [[%s.md|%s]]\n", projectName, projectName)

	return nil
}

func parseTasks(projectName string) ([]Task, error) {
	projectPath, err := getProjectPath(projectName)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(projectPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var tasks []Task
	var currentTask *Task
	taskPattern := regexp.MustCompile(`^## (\d{4}-\d{2}-\d{2}) (.+)$`)
	subtaskPattern := regexp.MustCompile(`^- \[([ x])\] (\d{4}-\d{2}-\d{2}) (.+)$`)
	statusPattern := regexp.MustCompile(`^\*\*Status:\*\* (\w+)`)
	completedPattern := regexp.MustCompile(`^\*\*Completed:\*\* (\d{4}-\d{2}-\d{2})?`)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	taskID := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		if match := taskPattern.FindStringSubmatch(line); match != nil {
			if currentTask != nil {
				tasks = append(tasks, *currentTask)
			}
			taskID++
			createdDate := match[1]
			name := match[2]
			currentTask = &Task{
				ID:          taskID,
				Name:        name,
				CreatedDate: createdDate,
				Status:      "open",
				LineNumber:  lineNum,
			}
			continue
		}

		if currentTask != nil {
			if match := statusPattern.FindStringSubmatch(line); match != nil {
				currentTask.Status = match[1]
				continue
			}
			if match := completedPattern.FindStringSubmatch(line); match != nil && match[1] != "" && len(currentTask.Subtasks) > 0 {
				currentTask.Subtasks[len(currentTask.Subtasks)-1].CompletedDate = match[1]
				continue
			}
			if match := completedPattern.FindStringSubmatch(line); match != nil && match[1] != "" {
				currentTask.CompletedDate = match[1]
				continue
			}
			if match := subtaskPattern.FindStringSubmatch(line); match != nil {
				status := "open"
				if match[1] == "x" {
					status = "closed"
				}
				subtask := Subtask{
					ID:          len(currentTask.Subtasks) + 1,
					CreatedDate: match[2],
					Description: match[3],
					Status:      status,
					LineNumber:  lineNum,
				}
				currentTask.Subtasks = append(currentTask.Subtasks, subtask)
			}
		}
	}

	if currentTask != nil {
		tasks = append(tasks, *currentTask)
	}

	return tasks, nil
}

func listTasks(projectName string, openOnly bool) error {
	tasks, err := parseTasks(projectName)
	if err != nil {
		return err
	}

	fmt.Printf("Tasks for project: %s\n\n", openStyle.Render(projectName))
	for _, task := range tasks {
		if openOnly && task.Status == "closed" {
			continue
		}

		statusIcon := openIcon
		if task.Status == "closed" {
			statusIcon = closedIcon
		}
		fmt.Printf("Task %2s: %s %s (%s", taskStyle.Render(strconv.Itoa(task.ID)), statusIcon, nameStyle.Render(task.Name), task.CreatedDate)
		if task.CompletedDate != "" {
			fmt.Printf(" -> %s", task.CompletedDate)
		}
		fmt.Printf(")\n")

		for _, subtask := range task.Subtasks {
			if openOnly && subtask.Status == "closed" {
				continue
			}
			subtaskIcon := openIcon
			if subtask.Status == "closed" {
				subtaskIcon = closedIcon
			}
			id := taskStyle.Render(fmt.Sprintf("%2d.%-2d", task.ID, subtask.ID))
			fmt.Printf("  - %s: %s %s (%s", id, subtaskIcon, nameStyle.Render(subtask.Description), subtask.CreatedDate)
			if subtask.CompletedDate != "" {
				fmt.Printf(" -> %s", subtask.CompletedDate)
			}
			fmt.Printf(")\n")
		}
	}
	return nil
}

func openTask(projectName, taskName, parentTaskID string) error {
	projectPath, err := getProjectPath(projectName)
	if err != nil {
		return err
	}

	today := time.Now().Format(dateFormat)

	if parentTaskID != "" {
		tasks, err := parseTasks(projectName)
		if err != nil {
			return err
		}

		var taskID int
		fmt.Sscanf(parentTaskID, "%d", &taskID)

		content, err := os.ReadFile(projectPath)
		if err != nil {
			return err
		}
		lines := strings.Split(string(content), "\n")

		for _, task := range tasks {
			if task.ID == taskID {
				taskLineIndex := -1
				for i, line := range lines {
					if strings.Contains(line, fmt.Sprintf("## %s %s", task.CreatedDate, task.Name)) {
						taskLineIndex = i
						break
					}
				}

				if taskLineIndex >= 0 {
					insertIndex := taskLineIndex + 1
					for insertIndex < len(lines) && !strings.HasPrefix(lines[insertIndex], "## ") {
						insertIndex++
					}
					newLines := make([]string, len(lines)+1)
					copy(newLines[:insertIndex], lines[:insertIndex])
					newLines[insertIndex] = fmt.Sprintf("- [ ] %s %s", today, taskName)
					copy(newLines[insertIndex+1:], lines[insertIndex:])
					return os.WriteFile(projectPath, []byte(strings.Join(newLines, "\n")), 0644)
				}
			}
		}
		return fmt.Errorf("task with ID %d not found", taskID)
	}

	file, err := os.OpenFile(projectPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = fmt.Fprintf(file, "\n## %s %s\n", today, taskName)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(file, "**Status:** open\n")
	return err
}

func closeTask(projectName, taskID string) error {
	tasks, err := parseTasks(projectName)
	if err != nil {
		return err
	}

	var targetTaskID int
	hasSubtaskID := false
	var parentTaskID int
	var subtaskID int

	if strings.Contains(taskID, ".") {
		parts := strings.Split(taskID, ".")
		fmt.Sscanf(parts[0], "%d", &parentTaskID)
		fmt.Sscanf(parts[1], "%d", &subtaskID)
		hasSubtaskID = true
	} else {
		fmt.Sscanf(taskID, "%d", &targetTaskID)
	}

	projectPath, err := getProjectPath(projectName)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(projectPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	today := time.Now().Format(dateFormat)

	if hasSubtaskID {
		for _, task := range tasks {
			if task.ID == parentTaskID {
				for i, line := range lines {
					for j, subtask := range task.Subtasks {
						if subtask.ID == subtaskID {
							re := regexp.MustCompile(fmt.Sprintf(`^- \[ \] (%s) (%s)$`, regexp.QuoteMeta(subtask.CreatedDate), regexp.QuoteMeta(subtask.Description)))
							if re.MatchString(line) {
								lines[i] = strings.Replace(line, "- [ ]", "- [x]", 1)
								newLines := make([]string, len(lines)+1)
								copy(newLines[:i+1], lines[:i+1])
								newLines[i+1] = fmt.Sprintf("**Completed:** %s", today)
								copy(newLines[i+2:], lines[i+1:])
								lines = newLines
								break
							}
							task.Subtasks[j].CompletedDate = today
						}
					}
				}
				break
			}
		}
	} else {
		var taskStartLine int
		var taskEndLine int

		for _, task := range tasks {
			if task.ID == targetTaskID {
				for i, line := range lines {
					if strings.Contains(line, fmt.Sprintf("## %s %s", task.CreatedDate, task.Name)) {
						taskStartLine = i
						break
					}
				}

				for j := taskStartLine + 1; j < len(lines); j++ {
					if strings.HasPrefix(lines[j], "## ") {
						taskEndLine = j
						break
					}
				}
				if taskEndLine == 0 {
					taskEndLine = len(lines)
				}

				for i := taskStartLine; i < taskEndLine; i++ {
					if strings.Contains(lines[i], "**Status:**") {
						lines[i] = "**Status:** closed"
					}
					if strings.Contains(lines[i], "**Completed:**") && !strings.HasPrefix(lines[i], "**Completed:**") {
						continue
					}
					if strings.Contains(lines[i], "- [ ]") {
						lines[i] = strings.Replace(lines[i], "- [ ]", "- [x]", 1)
						if !strings.Contains(lines[i], "Completed:") {
							lines[i] = lines[i] + fmt.Sprintf(" (Completed: %s)", today)
						}
					}
				}

				hasCompleted := false
				hasStatus := false
				for i := taskStartLine; i < taskEndLine; i++ {
					if strings.Contains(lines[i], "**Completed:**") {
						lines[i] = fmt.Sprintf("**Completed:** %s", today)
						hasCompleted = true
					}
					if strings.Contains(lines[i], "**Status:**") {
						lines[i] = "**Status:** closed"
						hasStatus = true
					}
				}

				if !hasCompleted && hasStatus {
					for i := taskStartLine; i < taskEndLine; i++ {
						if strings.Contains(lines[i], "**Status:** closed") {
							newLines := make([]string, len(lines)+1)
							copy(newLines[:i+1], lines[:i+1])
							newLines[i+1] = fmt.Sprintf("**Completed:** %s", today)
							copy(newLines[i+2:], lines[i+1:])
							lines = newLines
							break
						}
					}
				}

				break
			}
		}
	}

	return os.WriteFile(projectPath, []byte(strings.Join(lines, "\n")), 0644)
}

func clearCompleted(projectName string) error {
	tasks, err := parseTasks(projectName)
	if err != nil {
		return err
	}

	projectPath, err := getProjectPath(projectName)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(projectPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	i := 0
	for i < len(lines) {
		line := lines[i]
		isTaskLine := false
		taskEndLine := len(lines)

		for _, task := range tasks {
			if strings.Contains(line, fmt.Sprintf("## %s %s", task.CreatedDate, task.Name)) {
				if task.Status == "closed" {
					isTaskLine = true
					for j := i + 1; j < len(lines); j++ {
						if strings.HasPrefix(lines[j], "## ") {
							taskEndLine = j
							break
						}
					}
					break
				}
			}
		}

		if isTaskLine {
			i = taskEndLine
			continue
		}

		subtaskCompletedLine := false
		for _, task := range tasks {
			for _, subtask := range task.Subtasks {
				if subtask.Status == "closed" {
					re := regexp.MustCompile(fmt.Sprintf(`^- \[x\] (%s) (%s)$`, regexp.QuoteMeta(subtask.CreatedDate), regexp.QuoteMeta(subtask.Description)))
					if re.MatchString(line) {
						subtaskCompletedLine = true
						break
					}
					if strings.Contains(line, fmt.Sprintf("**Completed:** %s", subtask.CompletedDate)) {
						subtaskCompletedLine = true
						break
					}
				}
			}
			if subtaskCompletedLine {
				break
			}
		}

		if subtaskCompletedLine {
			i++
			continue
		}

		newLines = append(newLines, line)
		i++
	}

	return os.WriteFile(projectPath, []byte(strings.Join(newLines, "\n")), 0644)
}

func setupProject(projectName string) error {
	err := ensureTasksDir()
	if err != nil {
		return fmt.Errorf("error creating tasks directory: %v", err)
	}

	err = ensureIndexFile()
	if err != nil {
		return fmt.Errorf("error creating index file: %v", err)
	}

	err = ensureProjectFile(projectName)
	if err != nil {
		return fmt.Errorf("error creating project file: %v", err)
	}

	err = ensureLinkInIndex(projectName)
	if err != nil {
		return fmt.Errorf("error updating index: %v", err)
	}

	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "taskmanager",
		Short: "A simple task manager for vimwiki markdown files",
	}

	lsCmd := &cobra.Command{
		Use:   "ls [project]",
		Short: "List open tasks and subtasks",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			project := args[0]
			if err := setupProject(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := listTasks(project, true); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	allCmd := &cobra.Command{
		Use:   "all [project]",
		Short: "List all tasks and subtasks",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			project := args[0]
			if err := setupProject(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := listTasks(project, false); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	openCmd := &cobra.Command{
		Use:   "open [project] [name]",
		Short: "Open a new task or subtask",
		Args:  cobra.RangeArgs(2, 3),
		Run: func(cmd *cobra.Command, args []string) {
			project := args[0]
			name := args[1]
			parent, _ := cmd.Flags().GetString("parent")

			if err := setupProject(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := openTask(project, name, parent); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			_ = listTasks(project, true)
		},
	}
	openCmd.Flags().StringP("parent", "p", "", "Parent task ID")

	closeCmd := &cobra.Command{
		Use:   "close [project] [id]",
		Short: "Close an existing task or subtask",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			project := args[0]
			id := args[1]

			if err := setupProject(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := closeTask(project, id); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	clearCmd := &cobra.Command{
		Use:   "clear [project]",
		Short: "Delete all completed tasks and subtasks",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			project := args[0]

			if err := setupProject(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			if err := clearCompleted(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Cleared all completed tasks and subtasks from %s\n", project)
		},
	}

	rootCmd.AddCommand(lsCmd)
	rootCmd.AddCommand(allCmd)
	rootCmd.AddCommand(openCmd)
	rootCmd.AddCommand(closeCmd)
	rootCmd.AddCommand(clearCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
