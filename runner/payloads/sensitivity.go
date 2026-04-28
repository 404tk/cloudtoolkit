package payloads

type Sensitivity struct {
	Level      string
	ConfirmKey string
	Resource   string
}

func (s Sensitivity) RequiresConfirmation() bool {
	return s.ConfirmKey != ""
}

type SensitivePayload interface {
	Sensitivity(metadata string) Sensitivity
}

func DescribeSensitivity(name, metadata string) Sensitivity {
	payload, _, ok := Lookup(name)
	if !ok {
		return Sensitivity{}
	}
	sensitive, ok := payload.(SensitivePayload)
	if !ok {
		return Sensitivity{}
	}
	return sensitive.Sensitivity(metadata)
}
