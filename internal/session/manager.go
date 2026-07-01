package session

import "sync"

const serverVersion = "dev"

type Manager struct {
	mu    sync.RWMutex
	state State
}

type State struct {
	ActiveMap     *MapState `json:"activeMap"`
	Mask          MaskState `json:"mask"`
	ServerVersion string    `json:"serverVersion"`
}

type MapState struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type MaskState struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

func NewManager(initialMap *MapState) *Manager {
	return &Manager{
		state: State{
			ActiveMap: cloneMapState(initialMap),
			Mask: MaskState{
				Width:  512,
				Height: 512,
			},
			ServerVersion: serverVersion,
		},
	}
}

func (m *Manager) State() State {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.state
}

func (m *Manager) SetActiveMap(activeMap *MapState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.state.ActiveMap = cloneMapState(activeMap)
}

func cloneMapState(input *MapState) *MapState {
	if input == nil {
		return nil
	}

	copy := *input
	return &copy
}
