package application

import "seed-vigor-gate/internal/domain"

type WriteMeta struct {
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"idempotencyKey"`
	Actor           string `json:"actor"`
}
type CreateCaseCommand struct {
	IdempotencyKey    string `json:"idempotencyKey"`
	Actor             string `json:"actor"`
	AccessionCode     string `json:"accessionCode"`
	Source            string `json:"source"`
	HarvestedAt       string `json:"harvestedAt"`
	DeclaredSeedCount int    `json:"declaredSeedCount"`
	ProtocolCode      string `json:"protocolCode"`
}
type ConfirmSamplingCommand struct {
	WriteMeta
	SampleUnits       []string                 `json:"sampleUnits"`
	UnitQuotas        []domain.SampleUnitQuota `json:"unitQuotas"`
	ReplicateCount    int                      `json:"replicateCount"`
	SeedsPerReplicate int                      `json:"seedsPerReplicate"`
	TemperatureMinC   float64                  `json:"temperatureMinC"`
	TemperatureMaxC   float64                  `json:"temperatureMaxC"`
	EnvironmentNotes  string                   `json:"environmentNotes"`
}
type RecordObservationCommand struct {
	WriteMeta
	Rows                []ObservationRowCommand `json:"rows"`
	ReplicateNo         int                     `json:"replicateNo"`
	DayNo               int                     `json:"dayNo"`
	NormalCount         int                     `json:"normalCount"`
	AbnormalCount       int                     `json:"abnormalCount"`
	UngerminatedCount   int                     `json:"ungerminatedCount"`
	ContaminatedCount   int                     `json:"contaminatedCount"`
	TemperatureC        float64                 `json:"temperatureC"`
	Submitted           bool                    `json:"submitted"`
	SupplementalTrialID string                  `json:"supplementalTrialId"`
}
type ObservationRowCommand struct {
	ReplicateNo         int     `json:"replicateNo"`
	DayNo               int     `json:"dayNo"`
	NormalCount         int     `json:"normalCount"`
	AbnormalCount       int     `json:"abnormalCount"`
	UngerminatedCount   int     `json:"ungerminatedCount"`
	ContaminatedCount   int     `json:"contaminatedCount"`
	TemperatureC        float64 `json:"temperatureC"`
	Submitted           bool    `json:"submitted"`
	SupplementalTrialID string  `json:"supplementalTrialId,omitempty"`
}
type AnalyzeCommand struct {
	WriteMeta
	SupplementalTrialID string `json:"supplementalTrialId"`
}
type ResolveDeviationCommand struct {
	WriteMeta
	Cause                  string `json:"cause"`
	CorrectiveAction       string `json:"correctiveAction"`
	SupplementalTrialID    string `json:"supplementalTrialId"`
	StartSupplementalTrial bool   `json:"startSupplementalTrial"`
}
type ReviewCommand struct {
	WriteMeta
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}
type FreezeCommand struct{ WriteMeta }
type IssueCredentialCommand struct{ WriteMeta }

type WorkbenchCase struct {
	Case        *domain.QualificationCase `json:"case"`
	Timeline    []domain.AuditEvent       `json:"timeline"`
	NextActions []string                  `json:"nextActions"`
}
type ObservationRevisionResult struct {
	ReplicateNo         int    `json:"replicateNo"`
	DayNo               int    `json:"dayNo"`
	Revision            int    `json:"revision"`
	ObservationID       string `json:"observationId"`
	SupplementalTrialID string `json:"supplementalTrialId,omitempty"`
}
type ObservationBatchResult struct {
	AcceptedRows   int                         `json:"acceptedRows"`
	DraftCount     int                         `json:"draftCount"`
	SubmittedCount int                         `json:"submittedCount"`
	Version        int64                       `json:"version"`
	Status         domain.Status               `json:"status"`
	Revisions      []ObservationRevisionResult `json:"revisions"`
}
type VerificationCheck struct {
	Code         string   `json:"code"`
	Passed       bool     `json:"passed"`
	Message      string   `json:"message"`
	EvidenceRefs []string `json:"evidenceRefs"`
}
type CredentialVerification struct {
	Valid              bool                          `json:"valid"`
	Credential         *domain.EligibilityCredential `json:"credential"`
	Bundle             *domain.EvidenceBundle        `json:"evidenceBundle"`
	Timeline           []domain.AuditEvent           `json:"timeline"`
	Message            string                        `json:"message"`
	StoredDigest       string                        `json:"storedDigest"`
	RecalculatedDigest string                        `json:"recalculatedDigest"`
	Checks             []VerificationCheck           `json:"checks"`
}
