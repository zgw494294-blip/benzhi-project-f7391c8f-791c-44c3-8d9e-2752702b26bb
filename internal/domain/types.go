package domain

import "time"

type Status string

const (
	StatusDraft             Status = "draft"
	StatusSamplingConfirmed Status = "sampling_confirmed"
	StatusObserving         Status = "observing"
	StatusPendingReview     Status = "pending_review"
	StatusReturned          Status = "returned"
	StatusApproved          Status = "approved"
	StatusFrozen            Status = "frozen"
	StatusCredentialIssued  Status = "credential_issued"
)

type QualificationCase struct {
	ID                 string                 `json:"id"`
	AccessionCode      string                 `json:"accessionCode"`
	Source             string                 `json:"source"`
	HarvestedAt        string                 `json:"harvestedAt"`
	DeclaredSeedCount  int                    `json:"declaredSeedCount"`
	ProtocolCode       string                 `json:"protocolCode"`
	Status             Status                 `json:"status"`
	Version            int64                  `json:"version"`
	CreatedAt          time.Time              `json:"createdAt"`
	UpdatedAt          time.Time              `json:"updatedAt"`
	SamplingPlan       *SamplingPlan          `json:"samplingPlan,omitempty"`
	Observations       []ObservationRevision  `json:"observations"`
	SupplementalTrials []SupplementalTrial    `json:"supplementalTrials"`
	Analysis           *AnalysisSnapshot      `json:"analysis,omitempty"`
	AnalysisSnapshots  []AnalysisSnapshot     `json:"analysisSnapshots"`
	Deviations         []Deviation            `json:"deviations"`
	Reviews            []ReviewDecision       `json:"reviews"`
	EvidenceBundle     *EvidenceBundle        `json:"evidenceBundle,omitempty"`
	Credential         *EligibilityCredential `json:"credential,omitempty"`
}

type SamplingPlan struct {
	ID                string            `json:"id"`
	CaseID            string            `json:"caseId"`
	Revision          int               `json:"revision"`
	SampleUnits       []string          `json:"sampleUnits"`
	UnitQuotas        []SampleUnitQuota `json:"unitQuotas"`
	ReplicateCount    int               `json:"replicateCount"`
	SeedsPerReplicate int               `json:"seedsPerReplicate"`
	TemperatureMinC   float64           `json:"temperatureMinC"`
	TemperatureMaxC   float64           `json:"temperatureMaxC"`
	EnvironmentNotes  string            `json:"environmentNotes"`
	TotalSampled      int               `json:"totalSampled"`
	SamplingRatio     float64           `json:"samplingRatio"`
	RemainingReserve  int               `json:"remainingReserve"`
	ConfirmedBy       string            `json:"confirmedBy"`
	ConfirmedAt       *time.Time        `json:"confirmedAt,omitempty"`
}

type SampleUnitQuota struct {
	UnitName     string `json:"unitName"`
	PlannedCount int    `json:"plannedCount"`
	ReplicateNos []int  `json:"replicateNos"`
}

type ObservationRevision struct {
	ID                  string    `json:"id"`
	CaseID              string    `json:"caseId"`
	ReplicateNo         int       `json:"replicateNo"`
	DayNo               int       `json:"dayNo"`
	Revision            int       `json:"revision"`
	NormalCount         int       `json:"normalCount"`
	AbnormalCount       int       `json:"abnormalCount"`
	UngerminatedCount   int       `json:"ungerminatedCount"`
	ContaminatedCount   int       `json:"contaminatedCount"`
	TemperatureC        float64   `json:"temperatureC"`
	RecordedBy          string    `json:"recordedBy"`
	RecordedAt          time.Time `json:"recordedAt"`
	Submitted           bool      `json:"submitted"`
	SupplementalTrialID string    `json:"supplementalTrialId,omitempty"`
	BatchID             string    `json:"batchId,omitempty"`
}

type Metric struct {
	EffectiveSampleSize int     `json:"effectiveSampleSize"`
	NormalTotal         int     `json:"normalTotal"`
	GerminationRate     float64 `json:"germinationRate"`
	ReplicateDifference float64 `json:"replicateDifference"`
	CandidateDecision   string  `json:"candidateDecision"`
}
type Finding struct {
	RuleCode     string   `json:"ruleCode"`
	Severity     string   `json:"severity"`
	Message      string   `json:"message"`
	EvidenceRefs []string `json:"evidenceRefs"`
}
type AnalysisSnapshot struct {
	ID                  string                  `json:"id"`
	CaseID              string                  `json:"caseId"`
	Sequence            int                     `json:"sequence"`
	CalculatedAt        time.Time               `json:"calculatedAt"`
	Metric              Metric                  `json:"metric"`
	Findings            []Finding               `json:"findings"`
	Trajectories        []GerminationTrajectory `json:"trajectories"`
	SupplementalTrialID string                  `json:"supplementalTrialId,omitempty"`
}

type GerminationTrajectory struct {
	ReplicateNo      int     `json:"replicateNo"`
	DayNo            int     `json:"dayNo"`
	ObservationRef   string  `json:"observationRef"`
	GerminationRate  float64 `json:"germinationRate"`
	AdjacentDayDelta float64 `json:"adjacentDayDelta"`
	PlateauLength    int     `json:"plateauLength"`
}

type Deviation struct {
	ID                  string    `json:"id"`
	CaseID              string    `json:"caseId"`
	RuleCode            string    `json:"ruleCode"`
	Severity            string    `json:"severity"`
	Status              string    `json:"status"`
	EvidenceRefs        []string  `json:"evidenceRefs"`
	Cause               string    `json:"cause,omitempty"`
	CorrectiveAction    string    `json:"correctiveAction,omitempty"`
	SupplementalTrialID string    `json:"supplementalTrialId,omitempty"`
	ResolvedBy          string    `json:"resolvedBy,omitempty"`
	ResolvedAt          time.Time `json:"resolvedAt,omitempty"`
	ClosedAnalysisID    string    `json:"closedAnalysisId,omitempty"`
}

type SupplementalTrial struct {
	ID                string                `json:"id"`
	CaseID            string                `json:"caseId"`
	DeviationID       string                `json:"deviationId"`
	SamplingPlanID    string                `json:"samplingPlanId"`
	InitiatedBy       string                `json:"initiatedBy"`
	InitiatedAt       time.Time             `json:"initiatedAt"`
	Status            string                `json:"status"`
	Observations      []ObservationRevision `json:"observations"`
	Analysis          *AnalysisSnapshot     `json:"analysis,omitempty"`
	AnalysisSnapshots []AnalysisSnapshot    `json:"analysisSnapshots"`
	CompletedAt       *time.Time            `json:"completedAt,omitempty"`
}

type ReviewDecision struct {
	ID          string    `json:"id"`
	Decision    string    `json:"decision"`
	Reason      string    `json:"reason"`
	Reviewer    string    `json:"reviewer"`
	CaseVersion int64     `json:"caseVersion"`
	AnalysisID  string    `json:"analysisId"`
	DecidedAt   time.Time `json:"decidedAt"`
}
type EvidenceBundle struct {
	ID                 string                `json:"id"`
	CaseID             string                `json:"caseId"`
	CaseVersion        int64                 `json:"caseVersion"`
	ProtocolCode       string                `json:"protocolCode"`
	SamplingPlan       SamplingPlan          `json:"samplingPlan"`
	Observations       []ObservationRevision `json:"observations"`
	Analysis           AnalysisSnapshot      `json:"analysis"`
	Deviations         []Deviation           `json:"deviations"`
	Reviews            []ReviewDecision      `json:"reviews"`
	SupplementalTrials []SupplementalTrial   `json:"supplementalTrials"`
	EvidenceRefs       []string              `json:"evidenceRefs"`
	Digest             string                `json:"digest"`
	FrozenBy           string                `json:"frozenBy"`
	FrozenAt           time.Time             `json:"frozenAt"`
}
type EligibilityCredential struct {
	CredentialNo     string    `json:"credentialNo"`
	CaseID           string    `json:"caseId"`
	EvidenceBundleID string    `json:"evidenceBundleId"`
	EvidenceDigest   string    `json:"evidenceDigest"`
	Decision         string    `json:"decision"`
	ProtocolCode     string    `json:"protocolCode"`
	IssuedBy         string    `json:"issuedBy"`
	IssuedAt         time.Time `json:"issuedAt"`
}
type AuditEvent struct {
	Sequence   int64          `json:"sequence"`
	CaseID     string         `json:"caseId"`
	Action     string         `json:"action"`
	Actor      string         `json:"actor"`
	OccurredAt time.Time      `json:"occurredAt"`
	Details    map[string]any `json:"details,omitempty"`
}
