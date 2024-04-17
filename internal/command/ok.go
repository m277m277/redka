package command

// Dummy command that always returns OK.
type OK struct {
	baseCmd
}

func parseOK(b baseCmd) (*OK, error) {
	return &OK{baseCmd: b}, nil
}

func (c *OK) Run(w Writer, _ Redka) (any, error) {
	w.WriteString("OK")
	return true, nil
}
