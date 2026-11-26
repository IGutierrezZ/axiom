package pipeline

import ()

type fixtureSkip string

func (s fixtureSkip) Error() string      { return string(s) }
func (s fixtureSkip) SkipReason() string { return string(s) }
