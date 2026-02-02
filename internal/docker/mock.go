package docker

// MockCommandExecutor is a test implementation that doesn't actually execute commands
type MockCommandExecutor struct {
	// Map of command to mock output
	MockOutputs map[string][]byte
	// Map of command to mock error
	MockErrors map[string]error
	// Track executed commands
	ExecutedCommands []CommandExecution
}

// CommandExecution records a command execution
type CommandExecution struct {
	Name string
	Args []string
	Dir  string
}

// NewMockCommandExecutor creates a new mock command executor
func NewMockCommandExecutor() *MockCommandExecutor {
	return &MockCommandExecutor{
		MockOutputs:      make(map[string][]byte),
		MockErrors:       make(map[string]error),
		ExecutedCommands: make([]CommandExecution, 0),
	}
}

// ExecuteCommand records the command and returns mocked output/error
func (m *MockCommandExecutor) ExecuteCommand(name string, args ...string) ([]byte, error) {
	return m.executeCommand("", name, args)
}

// ExecuteCommandInDir records the command and returns mocked output/error
func (m *MockCommandExecutor) ExecuteCommandInDir(dir, name string, args ...string) ([]byte, error) {
	return m.executeCommand(dir, name, args)
}

// executeCommand is the internal method that handles both execution types
func (m *MockCommandExecutor) executeCommand(dir, name string, args []string) ([]byte, error) {
	// Record the command execution
	m.ExecutedCommands = append(m.ExecutedCommands, CommandExecution{
		Name: name,
		Args: args,
		Dir:  dir,
	})

	// Create a key for looking up mocks
	key := name
	for _, arg := range args {
		key += " " + arg
	}

	// Return mocked error if available
	if err, exists := m.MockErrors[key]; exists {
		return nil, err
	}

	// Return mocked output if available
	if output, exists := m.MockOutputs[key]; exists {
		return output, nil
	}

	// Default behavior - return empty success
	return []byte("success"), nil
}

// SetMockOutput sets a mock output for a specific command
func (m *MockCommandExecutor) SetMockOutput(command string, args []string, output []byte) {
	key := command
	for _, arg := range args {
		key += " " + arg
	}
	m.MockOutputs[key] = output
}

// SetMockError sets a mock error for a specific command
func (m *MockCommandExecutor) SetMockError(command string, args []string, err error) {
	key := command
	for _, arg := range args {
		key += " " + arg
	}
	m.MockErrors[key] = err
}

// GetExecutedCommands returns all executed commands
func (m *MockCommandExecutor) GetExecutedCommands() []CommandExecution {
	return m.ExecutedCommands
}

// Clear clears all recorded commands and mocks
func (m *MockCommandExecutor) Clear() {
	m.ExecutedCommands = make([]CommandExecution, 0)
	m.MockOutputs = make(map[string][]byte)
	m.MockErrors = make(map[string]error)
}

// AssertCommandExecuted checks if a specific command was executed
func (m *MockCommandExecutor) AssertCommandExecuted(command string, args []string) bool {
	key := command
	for _, arg := range args {
		key += " " + arg
	}

	for _, execution := range m.ExecutedCommands {
		executionKey := execution.Name
		for _, arg := range execution.Args {
			executionKey += " " + arg
		}

		if executionKey == key {
			return true
		}
	}

	return false
}

// GetCommandCount returns the number of times a command was executed
func (m *MockCommandExecutor) GetCommandCount(command string, args []string) int {
	key := command
	for _, arg := range args {
		key += " " + arg
	}

	count := 0
	for _, execution := range m.ExecutedCommands {
		executionKey := execution.Name
		for _, arg := range execution.Args {
			executionKey += " " + arg
		}

		if executionKey == key {
			count++
		}
	}

	return count
}
