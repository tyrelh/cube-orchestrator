package task

import "slices"

type State int

const (
	Pending State = iota
	Scheduled
	Running
	Completed
	Failed
)

var stateTransitionMap = map[State][]State{
	Pending:   {Scheduled},
	Scheduled: {Scheduled, Running, Failed},
	Running:   {Running, Completed, Failed},
	Completed: {},
	Failed:    {Scheduled},
}

// func Contains(states []State, state State) bool {
// 	return slices.Contains(states, state)
// }

func ValidStateTransition(src State, dst State) bool {
	// return Contains(stateTransitionMap[src], dst)
	return slices.Contains(stateTransitionMap[src], dst)
}
