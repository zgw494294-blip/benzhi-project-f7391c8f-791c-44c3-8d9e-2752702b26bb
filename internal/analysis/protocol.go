package analysis

type Protocol struct {
	MinimumGerminationRate   float64
	MaximumReplicateSpread   float64
	MaximumContaminationRate float64
	MinimumObservationDay    int
}

func ProtocolFor(code string) Protocol {
	switch code {
	case "ISTA-2025", "GB-T3543-2025":
		return Protocol{MinimumGerminationRate: 85, MaximumReplicateSpread: 15, MaximumContaminationRate: 5, MinimumObservationDay: 7}
	default:
		return Protocol{MinimumGerminationRate: 80, MaximumReplicateSpread: 20, MaximumContaminationRate: 10, MinimumObservationDay: 7}
	}
}
