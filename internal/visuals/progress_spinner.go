package visuals

type ProgressSpinner struct {
	States       []string
	CurrentState int
}

func NewSpinner() *ProgressSpinner {
	return &ProgressSpinner{
		States: []string{"-", "\\", "|", "/"},
	}
}

func (sp *ProgressSpinner) Spin() {
	if sp.CurrentState+1 >= len(sp.States) {
		sp.CurrentState = 0
	} else {
		sp.CurrentState++
	}
}

func (sp *ProgressSpinner) Start() {
	sp.CurrentState = 0
}

func (sp *ProgressSpinner) Print() string {
	return sp.States[sp.CurrentState]
}
